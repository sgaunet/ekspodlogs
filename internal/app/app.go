package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"golang.org/x/time/rate"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/sirupsen/logrus"
)

// quota for AWS API: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const eventsRateLimit = 30
const logGroupRateLimit = 10

// App is the main structure of the application
type App struct {
	appLog               *logrus.Logger
	cfg                  aws.Config
	eventsRateLimit      *rate.Limiter
	logGroupRateLimit    *rate.Limiter
	clientCloudwatchlogs *cloudwatchlogs.Client
}

// New creates a new App
func New(cfg aws.Config) *App {
	er := rate.NewLimiter(rate.Every(1*time.Second), eventsRateLimit)
	lgr := rate.NewLimiter(rate.Every(1*time.Second), logGroupRateLimit)
	clientCloudwatchlogs := cloudwatchlogs.NewFromConfig(cfg)
	app := App{
		cfg:                  cfg,
		eventsRateLimit:      er,
		logGroupRateLimit:    lgr,
		clientCloudwatchlogs: clientCloudwatchlogs,
	}
	app.InitLog()
	return &app
}

// PrintID prints AWS identity only for debug purpose
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

// InitLog initializes the logger
func (a *App) InitLog() {
	appLog := logrus.New()
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})
	appLog.SetFormatter(&logrus.TextFormatter{
		DisableColors:    false,
		FullTimestamp:    false,
		DisableTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	appLog.SetOutput(os.Stdout)

	switch os.Getenv("DEBUGLEVEL") {
	case "info":
		appLog.SetLevel(logrus.InfoLevel)
	case "warn":
		appLog.SetLevel(logrus.WarnLevel)
	case "error":
		appLog.SetLevel(logrus.ErrorLevel)
	case "debug":
		appLog.SetLevel(logrus.DebugLevel)
	default:
		appLog.SetLevel(logrus.InfoLevel)
	}
	a.appLog = appLog
	a.appLog.Infoln("Log level:", a.appLog.Level)
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
		fmt.Printf("%s -- %s -- %s\n", timeT, lineOfLog.Kubernetes.ContainerName, lineOfLog.Log)
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

// FindLogGroupAuto finds the EKS log group automatically
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
	minTimeStampInMs := startTime.Unix() * 1000
	maxTimeStampInMs := endTime.Unix() * 1000

	logStreams, err := a.findLogStream(ctx, groupName, logStream, minTimeStampInMs, maxTimeStampInMs)
	if err != nil {
		return err
	}
	for _, l := range logStreams {
		// store events in sqlite
		err = a.getEvents(ctx, groupName, *l.LogStreamName, minTimeStampInMs, maxTimeStampInMs, "")
		if err != nil {
			return err
		}
	}
	return nil
}
