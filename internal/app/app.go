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

const eventsRateLimit = 20
const logGroupRateLimit = 5

type App struct {
	appLog            *logrus.Logger
	cfg               aws.Config
	eventsRateLimit   *rate.Limiter
	logGroupRateLimit *rate.Limiter
}

func New(cfg aws.Config) *App {
	er := rate.NewLimiter(rate.Every(1*time.Second), eventsRateLimit)
	lgr := rate.NewLimiter(rate.Every(1*time.Second), logGroupRateLimit)
	app := App{
		cfg:               cfg,
		eventsRateLimit:   er,
		logGroupRateLimit: lgr,
	}
	app.InitLog()
	return &app
}

// print AWS identity
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
func (a *App) getEvents(ctx context.Context, groupName string, streamName string, client *cloudwatchlogs.Client, minTimeStamp int64, maxTimeStamp int64, nextToken string) error {
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
	res, err := client.GetLogEvents(ctx, &input)
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
		return a.getEvents(ctx, groupName, streamName, client, minTimeStamp, maxTimeStamp, *res.NextForwardToken)
	}
	return nil
}

func (a *App) ListLogGroups(ctx context.Context, cfg aws.Config, NextToken string) error {
	clientCloudwatchlogs := cloudwatchlogs.NewFromConfig(cfg)
	loggroups, err := a.recurseListLogGroup(ctx, clientCloudwatchlogs, "")
	for i := range loggroups {
		a.appLog.Infoln(loggroups[i])
	}
	return err
}

func (a *App) FindLogGroupAuto(ctx context.Context, cfg aws.Config) (string, error) {
	clientCloudwatchlogs := cloudwatchlogs.NewFromConfig(cfg)
	loggroups, err := a.recurseListLogGroup(ctx, clientCloudwatchlogs, "")
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

func (a *App) PrintEvents(ctx context.Context, cfg aws.Config, groupName string, logStream string, startTime time.Time, endTime time.Time) error {
	clientCloudwatchlogs := cloudwatchlogs.NewFromConfig(cfg)
	minTimeStampInMs := startTime.Unix() * 1000
	maxTimeStampInMs := endTime.Unix() * 1000

	logStreams, err := a.findLogStream(ctx, clientCloudwatchlogs, groupName, logStream, minTimeStampInMs, maxTimeStampInMs)
	if err != nil {
		return err
	}
	for _, l := range logStreams {
		// store events in sqlite
		err = a.getEvents(context.Background(), groupName, *l.LogStreamName, clientCloudwatchlogs, minTimeStampInMs, maxTimeStampInMs, "")
		if err != nil {
			return err
		}
	}
	return nil
}
