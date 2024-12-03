package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/dromara/carbon/v2"
	"github.com/sirupsen/logrus"
)

// ConvertTimeToCarbon converts the begin and end date to carbon.Carbon
func ConvertTimeToCarbon(beginDate, endDate string) (carbon.Carbon, carbon.Carbon, error) {
	b := carbon.Parse(beginDate)
	if b.Error != nil {
		return carbon.Carbon{}, carbon.Carbon{}, fmt.Errorf("invalid begin date: %w", b.Error)
	}
	e := carbon.Parse(endDate)
	if e.Error != nil {
		return carbon.Carbon{}, carbon.Carbon{}, fmt.Errorf("invalid end date: %w", e.Error)
	}
	if b.Gt(e) {
		return carbon.Carbon{}, carbon.Carbon{}, errors.New("begin date is after end date")
	}
	return b, e, nil
}

// InitAWSConfig initializes the AWS SDK configuration
// If the ssoProfile is empty, it will use the default profile
func InitAWSConfig(ctx context.Context, profile string) (cfg aws.Config, err error) {
	if len(ssoProfile) == 0 {
		cfg, err = config.LoadDefaultConfig(ctx)
	} else {
		// Try to connect with the SSO profile put in parameter
		cfg, err = config.LoadDefaultConfig(
			ctx,
			config.WithSharedConfigProfile(ssoProfile),
		)
	}
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to load SDK config: %w", err)
	}
	return cfg, nil
}

// NewLogger creates a new logger
// The debugLevel is the variable to set the log level
// It can be "info", "warn", "error" or "debug"
// If the variable is not set, the default log level is "info"
// Be careful, the log level is case sensitive and by default,
// every logs are **discarded**, set the output to os.Stdout to see the logs
// Example: l:=NewLogger(); l.SetOutput(os.Stdout)
func NewLogger(debugLevel string) *logrus.Logger {
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
	appLog.SetOutput(io.Discard)

	switch os.Getenv(debugLevel) {
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
	appLog.Infoln("Log level:", appLog.Level)
	return appLog
}
