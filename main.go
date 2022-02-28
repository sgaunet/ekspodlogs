package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sgaunet/ekspodlogs/app"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func checkErrorAndExitIfErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}

var version string = "development"

func printVersion() {
	fmt.Println(version)
}

func main() {
	var cfg aws.Config // Configuration to connect to AWS API
	var vOption, listGroupOption bool
	var logStream, groupName, ssoProfile string
	var err error
	var lastPeriodToWatch int
	var startDate, endDate string
	var startTime, endTime time.Time

	// Treat args
	flag.BoolVar(&vOption, "v", false, "Get version")
	flag.BoolVar(&listGroupOption, "lg", false, "List LogGroup")
	flag.StringVar(&groupName, "g", "", "LogGroup to parse")
	flag.StringVar(&ssoProfile, "p", "", "Auth by SSO")
	flag.StringVar(&startDate, "s", "", "Start date (YYYY-MM-DD HH:MM:SS)")
	flag.StringVar(&endDate, "e", "", "End date  (YYYY-MM-DD HH:MM:SS)")
	flag.StringVar(&logStream, "l", "", "LogStream to search")
	flag.Parse()

	if vOption {
		printVersion()
		os.Exit(0)
	}

	// No profile selected
	if len(ssoProfile) == 0 {
		cfg, err = config.LoadDefaultConfig(context.TODO())
		checkErrorAndExitIfErr(err)
	} else {
		// Try to connect with the SSO profile put in parameter
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithSharedConfigProfile(ssoProfile),
		)
		checkErrorAndExitIfErr(err)
	}

	app := app.New(lastPeriodToWatch, cfg)
	if err = app.PrintID(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	// Option -lg to list loggroup : list and quit
	if listGroupOption {
		app.ListLogGroups(cfg, "")
		os.Exit(0)
	}

	// Get logs, controls parameters
	if len(logStream) == 0 {
		fmt.Fprintln(os.Stderr, "Mandatory option : -l")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(startDate) != 0 {
		startTime, err = time.Parse("2006-01-02 15:04", startDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot convert startDate: %v\n", err.Error())
			os.Exit(1)
		}
	}
	if len(endDate) != 0 {
		endTime, err = time.Parse("2006-01-02 15:04", endDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot convert endDate: %v\n", err.Error())
			os.Exit(1)
		}
	}

	err = app.FindLogStream(cfg, groupName, logStream, startTime, endTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logstream not found ? (%v)\n", err.Error())
		os.Exit(1)
	}
}
