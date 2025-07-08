package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/dromara/carbon/v2"
	"golang.org/x/sync/errgroup"
)

// Recursive function that will return true if the groupName parameter has been found or not
func (a *App) findLogGroup(ctx context.Context, groupName string, NextToken string) (bool, error) {
	var params cloudwatchlogs.DescribeLogGroupsInput

	if len(NextToken) != 0 {
		params.NextToken = &NextToken
	}
	res, err := a.clientCloudwatchlogs.DescribeLogGroups(ctx, &params)
	if err != nil {
		return false, fmt.Errorf("error while calling DescribeLogGroups: %w", err)
	}
	for _, i := range res.LogGroups {
		a.appLog.Debugf("## Parse Log Group Name : %s\n", *i.LogGroupName)
		if *i.LogGroupName == groupName {
			return true, nil
		}
	}
	if res.NextToken == nil {
		// No token given, end of potential recursive call to parse the list of loggroups
		return false, nil
	}
	return a.findLogGroup(ctx, groupName, *res.NextToken)
}


// recursive function to list on stdout tge loggroup
func (a *App) recurseListLogGroup(ctx context.Context, client *cloudwatchlogs.Client, NextToken string) (loggroups []string, err error) {
	return a.recurseListLogGroupWithDepth(ctx, client, NextToken, 0)
}

// recurseListLogGroupWithDepth handles pagination with depth limiting
func (a *App) recurseListLogGroupWithDepth(ctx context.Context, client *cloudwatchlogs.Client, NextToken string, depth int) (loggroups []string, err error) {
	var params cloudwatchlogs.DescribeLogGroupsInput
	if len(NextToken) != 0 {
		params.NextToken = &NextToken
	}
	if err := a.logGroupRateLimit.Wait(ctx); err != nil {
		return loggroups, fmt.Errorf("rate limit wait error: %w", err)
	}
	res, err := client.DescribeLogGroups(ctx, &params)
	if err != nil {
		return loggroups, err
	}
	for _, i := range res.LogGroups {
		// fmt.Printf("%s\n", *i.LogGroupName)
		loggroups = append(loggroups, *i.LogGroupName)
		// var glgfi cloudwatchlogs.GetLogGroupFieldsInput
		// glgfi.LogGroupName = i.LogGroupName

		// glgfo, _ := client.GetLogGroupFields(context.TODO(), &glgfi)
		// for _, logGrpF := range glgfo.LogGroupFields {
		// fmt.Println(*logGrpF.Name)
		// }
		// fmt.Println("")
	}
	if res.NextToken == nil {
		return loggroups, err
	} else {
		lg, err := a.recurseListLogGroupWithDepth(ctx, client, *res.NextToken, depth+1)
		loggroups = append(loggroups, lg...)
		return loggroups, err
	}
}

// function that parses every streams of loggroup groupName
func (a *App) findLogStream(ctx context.Context, groupName string, logStream string, minTimeStampInMs int64, maxTimeStampInMs int64) ([]types.LogStream, error) {
	doesGroupNameExists, err := a.findLogGroup(ctx, groupName, "")
	if err != nil {
		return nil, err
	}
	if !doesGroupNameExists {
		err := fmt.Errorf("GroupName %s not found", groupName)
		a.appLog.Errorln(err.Error())
		return nil, err
	}

	logstreams, err := a.parseAllStreamsOfGroupParallel(ctx, groupName, logStream, minTimeStampInMs, maxTimeStampInMs)
	return logstreams, err
}

// streamPage represents a page of log streams to fetch
type streamPage struct {
	nextToken string
	depth     int
}


// parseAllStreamsOfGroupParallel fetches log streams in parallel while respecting rate limits
func (a *App) parseAllStreamsOfGroupParallel(ctx context.Context, groupName string, logStream string, minTimeStamp int64, maxTimeStamp int64) ([]types.LogStream, error) {
	const maxConcurrentPages = 3 // Limit concurrent API calls to respect rate limits

	var allStreams []types.LogStream
	var streamsMutex sync.Mutex
	var shouldStopGlobal bool
	var stopMutex sync.Mutex
	
	// Channel to manage pages to process
	pagesChan := make(chan streamPage, 10)
	
	// Counter for pending work items
	var pendingWork int32
	var workMutex sync.Mutex
	var closeOnce sync.Once
	
	// Use errgroup for proper error handling
	g, ctx := errgroup.WithContext(ctx)
	
	// Function to check if we should stop
	checkShouldStop := func() bool {
		stopMutex.Lock()
		defer stopMutex.Unlock()
		return shouldStopGlobal
	}
	
	// Function to set global stop flag
	setStop := func() {
		stopMutex.Lock()
		defer stopMutex.Unlock()
		shouldStopGlobal = true
	}
	
	// Function to increment pending work
	incPendingWork := func() {
		workMutex.Lock()
		defer workMutex.Unlock()
		pendingWork++
	}
	
	// Function to decrement pending work and return count
	decPendingWork := func() int32 {
		workMutex.Lock()
		defer workMutex.Unlock()
		pendingWork--
		return pendingWork
	}
	
	// Function to safely send to channel
	safeSend := func(page streamPage) bool {
		select {
		case pagesChan <- page:
			incPendingWork()
			return true
		case <-ctx.Done():
			return false
		}
	}
	
	// Worker function to process pages
	processPage := func() error {
		for page := range pagesChan {
			// Rate limit the API call
			if err := a.logGroupRateLimit.Wait(ctx); err != nil {
				decPendingWork()
				return fmt.Errorf("rate limit wait error: %w", err)
			}
			
			// Prepare the API request
			var paramsLogStream cloudwatchlogs.DescribeLogStreamsInput
			paramsLogStream.LogGroupName = &groupName
			paramsLogStream.OrderBy = "LastEventTime"
			descending := true
			paramsLogStream.Descending = &descending
			// Set limit to maximum (50) to get as many streams as possible per API call
			limit := int32(50)
			paramsLogStream.Limit = &limit
			
			if len(page.nextToken) != 0 {
				paramsLogStream.NextToken = &page.nextToken
			}
			
			// Make the API call
			res, err := a.clientCloudwatchlogs.DescribeLogStreams(ctx, &paramsLogStream)
			if err != nil {
				decPendingWork()
				return fmt.Errorf("failed to describe log streams: %w", err)
			}
			
			// Process the streams from this page
			var pageStreams []types.LogStream
			var shouldStopLocal bool
			
			for _, j := range res.LogStreams {
				a.tui.IncNbLogStreams()
				
				// Check if stream matches the filter
				if strings.Contains(*j.LogStreamName, logStream) || len(logStream) == 0 {
					a.tui.IncNbLogStreamsFound()
					tm := time.Unix(*j.LastEventTimestamp/1000, 0)
					lastEvent := carbon.CreateFromTimestamp(*j.LastEventTimestamp / 1000)
					a.appLog.Debugf("Stream Name: %s\n", *j.LogStreamName)
					a.appLog.Debugf("LasteventTimeStamp: %d  (%s)\n", *j.LastEventTimestamp, lastEvent.ToDateTimeString())
					a.appLog.Debugf("Parse stream : %s (Last event %v)\n", *j.LogStreamName, tm)
					pageStreams = append(pageStreams, j)
				}
				
				// Check if we should stop processing older streams
				if *j.LastEventTimestamp < minTimeStamp {
					shouldStopLocal = true
					a.appLog.Debugf("%v < %v\n", *j.LastEventTimestamp, minTimeStamp)
					a.appLog.Debugf("%v < %v\n", time.Unix(*j.LastEventTimestamp/1000, 0), time.Unix(minTimeStamp/1000, 0))
					a.appLog.Debugln("stop to parse, *j.LastEventTimestamp < minTimeStamp")
					break
				}
			}
			
			// Add streams to the global collection
			streamsMutex.Lock()
			allStreams = append(allStreams, pageStreams...)
			streamsMutex.Unlock()
			
			// Set global stop flag if local conditions are met
			if shouldStopLocal {
				setStop()
			}
			
			// Check if we should continue with more pages
			if res.NextToken != nil && !shouldStopLocal && !checkShouldStop() {
				if !safeSend(streamPage{nextToken: *res.NextToken, depth: page.depth + 1}) {
					// safeSend only returns false if context is cancelled
					if ctx.Err() != nil {
						decPendingWork()
						return ctx.Err()
					}
				}
			}
			
			// Decrement pending work count
			remaining := decPendingWork()
			
			// If this was the last pending work item, close the channel
			if remaining == 0 {
				closeOnce.Do(func() {
					close(pagesChan)
				})
				return nil
			}
		}
		return nil
	}
	
	// Start workers
	for range maxConcurrentPages {
		g.Go(processPage)
	}
	
	// Start with the first page
	if !safeSend(streamPage{nextToken: "", depth: 0}) {
		return nil, ctx.Err()
	}
	
	// Wait for all workers to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}
	
	return allStreams, nil
}
