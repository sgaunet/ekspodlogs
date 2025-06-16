package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/dromara/carbon/v2"
)

// Recursive function that will return true if the groupName parameter has been found or not
func (a *App) findLogGroup(groupName string, NextToken string) (bool, error) {
	var params cloudwatchlogs.DescribeLogGroupsInput

	if len(NextToken) != 0 {
		params.NextToken = &NextToken
	}
	res, err := a.clientCloudwatchlogs.DescribeLogGroups(context.TODO(), &params)
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
	return a.findLogGroup(groupName, *res.NextToken)
}

// parseAllStreamsOfGroup parses every events of every streams of a group
// It's a recursive function
func (a *App) parseAllStreamsOfGroup(ctx context.Context, groupName string, logStream string, nextToken string, minTimeStamp int64, maxTimeStamp int64) ([]types.LogStream, error) {
	var paramsLogStream cloudwatchlogs.DescribeLogStreamsInput
	var stopToParseLogStream bool
	var logStreams []types.LogStream
	// Search logstreams of groupName
	// Ordered by last event time
	// descending
	paramsLogStream.LogGroupName = &groupName
	paramsLogStream.OrderBy = "LastEventTime"
	descending := true
	paramsLogStream.Descending = &descending

	if len(nextToken) != 0 {
		paramsLogStream.NextToken = &nextToken
	}
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs#Client.DescribeLogStreams
	// now := time.Now()
	if err := a.logGroupRateLimit.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait error: %w", err)
	}
	res2, err := a.clientCloudwatchlogs.DescribeLogStreams(context.TODO(), &paramsLogStream)
	if err != nil {
		return nil, err
	}
	// Loop over streams
	for _, j := range res2.LogStreams {
		a.tui.IncNbLogStreams()
		// fmt.Println(idx, *j.LogStreamName)
		if strings.Contains(*j.LogStreamName, logStream) || len(logStream) == 0 {
			a.tui.IncNbLogStreamsFound()
			tm := time.Unix(*j.LastEventTimestamp/1000, 0) // aws timestamp are in ms
			// convert tm to date
			lastEvent := carbon.CreateFromTimestamp(*j.LastEventTimestamp / 1000)
			a.appLog.Debugf("Stream Name: %s\n", *j.LogStreamName)
			a.appLog.Debugf("LasteventTimeStamp: %d  (%s)\n", *j.LastEventTimestamp, lastEvent.ToDateTimeString())
			a.appLog.Debugf("Parse stream : %s (Last event %v)\n", *j.LogStreamName, tm)
			logStreams = append(logStreams, j)
		}
		// No need to parse old logstream older than minTimeStamp
		if *j.LastEventTimestamp < minTimeStamp {
			stopToParseLogStream = true
			a.appLog.Debugf("%v < %v\n", *j.LastEventTimestamp, minTimeStamp)
			a.appLog.Debugf("%v < %v\n", time.Unix(*j.LastEventTimestamp/1000, 0), time.Unix(minTimeStamp/1000, 0))
			a.appLog.Debugln("stop to parse, *j.LastEventTimestamp < minTimeStamp")
			break
		}
	}

	if res2.NextToken != nil && !stopToParseLogStream {
		l, err := a.parseAllStreamsOfGroup(ctx, groupName, logStream, *res2.NextToken, minTimeStamp, maxTimeStamp)
		if err != nil {
			return nil, err
		}
		logStreams = append(logStreams, l...)
	}
	return logStreams, err
}

// recursive function to list on stdout tge loggroup
func (a *App) recurseListLogGroup(ctx context.Context, client *cloudwatchlogs.Client, NextToken string) (loggroups []string, err error) {
	var params cloudwatchlogs.DescribeLogGroupsInput
	if len(NextToken) != 0 {
		params.NextToken = &NextToken
	}
	if err := a.logGroupRateLimit.Wait(ctx); err != nil {
		return loggroups, fmt.Errorf("rate limit wait error: %w", err)
	}
	res, err := client.DescribeLogGroups(context.TODO(), &params)
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
		lg, err := a.recurseListLogGroup(ctx, client, *res.NextToken)
		loggroups = append(loggroups, lg...)
		return loggroups, err
	}
}

// function that parses every streams of loggroup groupName
func (a *App) findLogStream(ctx context.Context, groupName string, logStream string, minTimeStampInMs int64, maxTimeStampInMs int64) ([]types.LogStream, error) {
	doesGroupNameExists, err := a.findLogGroup(groupName, "")
	if err != nil {
		return nil, err
	}
	if !doesGroupNameExists {
		err := fmt.Errorf("GroupName %s not found", groupName)
		a.appLog.Errorln(err.Error())
		return nil, err
	}

	logstreams, err := a.parseAllStreamsOfGroup(ctx, groupName, logStream, "", minTimeStampInMs, maxTimeStampInMs)
	return logstreams, err
}
