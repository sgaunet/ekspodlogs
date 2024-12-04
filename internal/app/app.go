package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
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
const maxEventsAPICallPerSecond = 30
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
		return err
	}
	a.appLog.Debugf("Account: %s\n", aws.ToString(identity.Account))
	a.appLog.Debugf("UserID: %s\n", aws.ToString(identity.UserId))
	a.appLog.Debugf("ARN: %s\n", aws.ToString(identity.Arn))
	return nil
}

// getEvents parse events of a stream and print results that do not match with any rules on stdout
func (a *App) getEvents(ctx context.Context, groupName string, streamName string, minTimeStamp int64, maxTimeStamp int64, nextToken string) error {
	startFromHead := true
	input := cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &groupName,
		LogStreamName: &streamName,
		EndTime:       &maxTimeStamp,
		StartTime:     &minTimeStamp,
		StartFromHead: &startFromHead,
	}
	a.appLog.Debugf("maxTimeStamp=%v     //   %v\n", maxTimeStamp, time.Unix(maxTimeStamp/1000, 0))
	a.appLog.Debugf("minTimeStamp=%v     //   %v\n", minTimeStamp, time.Unix(minTimeStamp/1000, 0))

	if nextToken == "" {
		input.NextToken = nil
	} else {
		input.NextToken = &nextToken
	}

	a.appLog.Debugf("\n**Parse stream** : %s\n", streamName)
	a.eventsRateLimit.Wait(ctx)
	res, err := a.clientCloudwatchlogs.GetLogEvents(ctx, &input)
	if err != nil {
		return err
	}

	for _, k := range res.Events {
		var lineOfLog fluentDockerLog
		err := json.Unmarshal([]byte(*k.Message), &lineOfLog)
		if err != nil {
			a.appLog.Errorln(err.Error(), "Are you sure to parse logs of a container ? (done by fluentd)")
			return err
		}
		timeT := time.Unix(*k.Timestamp/1000, 0)
		// fmt.Printf("%s -- %s -- %s\n", timeT, lineOfLog.Kubernetes.ContainerName, lineOfLog.Log)
		err = a.queries.AddLog(ctx, a.profileName, groupName, timeT, lineOfLog.Kubernetes.PodName, lineOfLog.Kubernetes.ContainerName, lineOfLog.Kubernetes.NamespaceName, lineOfLog.Log)
		if err != nil {
			return err
		}
	}

	a.appLog.Debugln("             nextToken=", nextToken)
	a.appLog.Debugln(" *res.NextForwardToken=", *res.NextForwardToken)
	a.appLog.Debugln("*res.NextBackwardToken=", *res.NextBackwardToken)
	if *res.NextForwardToken != nextToken {
		return a.getEvents(ctx, groupName, streamName, minTimeStamp, maxTimeStamp, *res.NextForwardToken)
	}
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

// PrintEvents prints events of a log group
func (a *App) PrintEvents(ctx context.Context, groupName string, logStream string, startTime time.Time, endTime time.Time) error {
	var errWorker error
	minTimeStampInMs := startTime.Unix() * 1000
	maxTimeStampInMs := endTime.Unix() * 1000

	a.tui.StartSpinnerRetrieveLogStreams()
	logStreams, err := a.findLogStream(ctx, groupName, logStream, minTimeStampInMs, maxTimeStampInMs)
	if err != nil {
		// err spinner
		return err
	}
	a.tui.StopSpinnerRetrieveLogStreams()
	err = a.tui.StartSpinnerScanLogStreams()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error spinner: %s", err)
	}

	var wg sync.WaitGroup
	chWorker := make(chan workEvent, 3)
	// Launch workers in background
	wg.Add(1)
	go func() {
		errWorker = a.workerEvents(ctx, &wg, chWorker)
	}()

	for _, l := range logStreams {
		work := workEvent{
			groupName:    groupName,
			streamName:   *l.LogStreamName,
			minTimeStamp: minTimeStampInMs,
			maxTimeStamp: maxTimeStampInMs,
		}
		chWorker <- work
		a.tui.IncNbStreamsScanned()
		if err != nil {
			return err
		}
	}
	close(chWorker)
	wg.Wait()
	err = errWorker
	a.tui.StopSpinnerScanLogStreams()
	return err
}

// GetEvents returns events occured between two dates
// This function is used to get events from the database
func (a *App) GetEvents(ctx context.Context, profile string, groupName string, beginDate carbon.Carbon, endDate carbon.Carbon) ([]database.Log, error) {
	res, err := a.queries.GetLogs(ctx, beginDate, endDate, groupName, profile)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// workEvent is a struct to send work to workers
type workEvent struct {
	groupName    string
	streamName   string
	minTimeStamp int64
	maxTimeStamp int64
}

// workerEvents is a function to get events from a log stream
func (a *App) workerEvents(ctx context.Context, wg *sync.WaitGroup, work <-chan workEvent) error {
	var errGrp errgroup.Group
	var currentWorkers atomic.Int32
	var maxConcurrentWorkers int32 = 10

	for w := range work {
		for currentWorkers.Load() >= maxConcurrentWorkers {
			time.Sleep(1 * time.Second)
		}
		errGrp.Go(func() error {
			currentWorkers.Add(1)
			err := a.getEvents(ctx, w.groupName, w.streamName, w.minTimeStamp, w.maxTimeStamp, "")
			currentWorkers.Add(-1)
			return err
		})
	}
	err := errGrp.Wait()
	wg.Done()
	return err
}
