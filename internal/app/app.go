package app

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/dromara/carbon/v2"
	"github.com/sgaunet/ekspodlogs/internal/database"
	"github.com/sgaunet/ekspodlogs/pkg/storage/sqlite"
	"github.com/sgaunet/ekspodlogs/pkg/views"
	"github.com/sirupsen/logrus"
)

// quota for AWS API: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const maxEventsAPICallPerSecond = 5000
const maxLogGroupAPICALLPerSecond = 10

// App is the main structure of the application
type App struct {
	appLog               *logrus.Logger
	cfg                  aws.Config
	profileName          string
	eventsRateLimit      *rate.Limiter
	logGroupRateLimit    *rate.Limiter
	clientCloudwatchlogs *cloudwatchlogs.Client
	queries              *sqlite.Storage
	tui                  *views.TerminalView
}

// New creates a new App
func New(cfg aws.Config, profileName string, db *sqlite.Storage, tui *views.TerminalView) *App {
	clientCloudwatchlogs := cloudwatchlogs.NewFromConfig(cfg)
	app := App{
		cfg:                  cfg,
		profileName:          profileName,
		eventsRateLimit:      rate.NewLimiter(rate.Every(1*time.Second), maxEventsAPICallPerSecond),
		logGroupRateLimit:    rate.NewLimiter(rate.Every(1*time.Second), maxLogGroupAPICALLPerSecond),
		clientCloudwatchlogs: clientCloudwatchlogs,
		queries:              db,
		tui:                  tui,
		appLog:               logrus.New(),
	}
	return &app
}

// SetLogger sets the logger
func (a *App) SetLogger(logger *logrus.Logger) {
	a.appLog = logger
}

// PrintID prints AWS identity
// This function is used to test the AWS connection
// Set a logger at debug level to see the output
func (a *App) PrintID() error {
	client := sts.NewFromConfig(a.cfg)
	identity, err := client.GetCallerIdentity(
		context.Background(),
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}
	a.appLog.Debugf("Account: %s\n", aws.ToString(identity.Account))
	a.appLog.Debugf("UserID: %s\n", aws.ToString(identity.UserId))
	a.appLog.Debugf("ARN: %s\n", aws.ToString(identity.Arn))
	return nil
}



// ListLogGroup is a recursive function to list all log groups
func (a *App) ListLogGroups(ctx context.Context, NextToken string) error {
	loggroups, err := a.recurseListLogGroup(ctx, a.clientCloudwatchlogs, "")
	for i := range loggroups {
		a.appLog.Infoln(loggroups[i])
	}
	return err
}

// FindLogGroupAuto finds the EKS log group automatically by filtering the log groups
// It returns the log group name if only one is found
func (a *App) FindLogGroupAuto(ctx context.Context) (string, error) {
	loggroups, err := a.recurseListLogGroup(ctx, a.clientCloudwatchlogs, "")
	if err != nil {
		return "", err
	}

	var filteredLoggroups []string
	re := regexp.MustCompile(`/aws/containerinsights/.+/application`)
	for _, loggroup := range loggroups {
		if re.MatchString(loggroup) {
			filteredLoggroups = append(filteredLoggroups, loggroup)
		}
	}

	if len(filteredLoggroups) == 1 {
		return filteredLoggroups[0], err
	}
	return "", err
}

// PrintEvents prints events of a log group using FilterLogEvents for improved performance
func (a *App) PrintEvents(ctx context.Context, groupName string, logStream string, startTime time.Time, endTime time.Time) error {
	minTimeStampInMs := startTime.Unix() * 1000
	maxTimeStampInMs := endTime.Unix() * 1000

	// Add timeout to prevent indefinite hanging
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	a.tui.StartSpinnerRetrieveLogStreams()
	
	// Use FilterLogEvents instead of DescribeLogStreams + GetLogEvents for better performance
	err := a.processEventsWithFilter(ctx, groupName, logStream, minTimeStampInMs, maxTimeStampInMs)
	
	a.tui.StopSpinnerRetrieveLogStreams()
	if err != nil {
		return err
	}
	return nil
}

// processEventsWithFilter uses FilterLogEvents API to retrieve and process log events efficiently
func (a *App) processEventsWithFilter(ctx context.Context, groupName string, logStreamFilter string, minTimeStamp int64, maxTimeStamp int64) error {
	// Set up FilterLogEvents input parameters
	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &groupName,
		StartTime:    &minTimeStamp,
		EndTime:      &maxTimeStamp,
		Interleaved:  &[]bool{true}[0], // Sort events from multiple streams by timestamp
	}
	
	// Don't use LogStreamNamePrefix as EKS log stream names don't directly contain pod names
	// Instead, we'll filter by pod name at the application level after parsing the JSON
	a.appLog.Debugf("Note: Not using LogStreamNamePrefix filter, will filter by pod name in JSON content")
	
	// Create paginator for handling large result sets
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(a.clientCloudwatchlogs, input)
	
	eventCount := 0
	pageCount := 0
	filteredEventCount := 0
	
	a.appLog.Debugf("Starting FilterLogEvents for group %s with time range %d-%d", groupName, minTimeStamp, maxTimeStamp)
	if logStreamFilter != "" {
		a.appLog.Debugf("Will filter events by pod name containing: %s", logStreamFilter)
	}
	
	// Process all pages of results
	for paginator.HasMorePages() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Rate limit the API call
		if err := a.eventsRateLimit.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit wait error: %w", err)
		}
		
		// Get next page of events
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to filter log events: %w", err)
		}
		
		pageCount++
		a.appLog.Debugf("Processing page %d with %d events", pageCount, len(output.Events))
		
		// Update spinner with current progress
		a.tui.UpdateSpinnerRetrieveLogStreamsWithText(fmt.Sprintf("Processing events... page %d, %d events found", pageCount, eventCount))
		
		// Process events from this page
		for _, event := range output.Events {
			eventCount++
			
			// Parse the log message as fluentDockerLog
			var lineOfLog fluentDockerLog
			err := json.Unmarshal([]byte(*event.Message), &lineOfLog)
			if err != nil {
				// Log the error but continue processing other events
				a.appLog.Warnf("Failed to unmarshal log message (skipping): %v. Message: %s", err, *event.Message)
				continue
			}
			
			// Apply pod name filtering if specified (filter by actual pod name in parsed JSON)
			if logStreamFilter != "" && !strings.Contains(lineOfLog.Kubernetes.PodName, logStreamFilter) {
				continue
			}
			
			filteredEventCount++
			
			// Convert timestamp and save to database
			timeT := time.Unix(*event.Timestamp/1000, 0).UTC()
			err = a.queries.AddLog(ctx, a.profileName, groupName, timeT, 
				lineOfLog.Kubernetes.PodName, lineOfLog.Kubernetes.ContainerName, 
				lineOfLog.Kubernetes.NamespaceName, lineOfLog.Log)
			if err != nil {
				return fmt.Errorf("failed to add log: %w", err)
			}
			
			// Update spinner periodically to show progress
			if filteredEventCount%100 == 0 { // Update UI every 100 events to avoid too frequent updates
				a.tui.UpdateSpinnerRetrieveLogStreamsWithText(fmt.Sprintf("Processing events... %d saved to database", filteredEventCount))
			}
		}
	}
	
	a.appLog.Debugf("Completed FilterLogEvents processing: %d total events, %d matching filter, from %d pages", eventCount, filteredEventCount, pageCount)
	return nil
}

// GetEvents returns events occured between two dates
// This function is used to get events from the database
func (a *App) GetEvents(ctx context.Context, profile string, groupName string, podName string, beginDate *carbon.Carbon, endDate *carbon.Carbon) ([]database.Log, error) {
	res, err := a.queries.GetLogs(ctx, groupName, profile, podName, beginDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return res, nil
}
