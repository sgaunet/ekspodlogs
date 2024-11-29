package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/dromara/carbon/v2"
	"github.com/sgaunet/ekspodlogs/internal/app"
	"github.com/sgaunet/ekspodlogs/pkg/storage/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var beginDate string
var endDate string
var groupName string
var ssoProfile string
var logStream string

// flag.BoolVar(&listGroupOption, "lg", false, "List LogGroup")
// flag.StringVar(&groupName, "g", "", "LogGroup to parse (not mandatory if there is only one log group : /aws/containerinsights/<Name of your cluster>/application)")
// flag.StringVar(&ssoProfile, "p", "", "Auth by SSO")
// flag.StringVar(&startDate, "s", "", "Start date (YYYY-MM-DD HH:MM:SS) - mandatory")
// flag.StringVar(&endDate, "e", "", "End date  (YYYY-MM-DD HH:MM:SS) - mandatory")
// flag.StringVar(&logStream, "l", "", "LogStream to search - mandatory")

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "synchronise the local database with the logs of cloudwatch",
	Long:  `synchronise the local database with the logs of cloudwatch`,
	Run: func(cmd *cobra.Command, args []string) {
		var cfg aws.Config // Configuration to connect to AWS API
		var err error

		ctx := context.Background()
		fmt.Println("group:", groupName)
		fmt.Println("profile:", ssoProfile)
		fmt.Println("logstream:", logStream)
		fmt.Println("begin:", beginDate)
		fmt.Println("end:", endDate)

		b := carbon.Parse(beginDate)
		if b.Error != nil {
			fmt.Fprintln(os.Stderr, "Invalid begin date")
			os.Exit(1)
		}
		e := carbon.Parse(endDate)
		if e.Error != nil {
			fmt.Fprintln(os.Stderr, "Invalid end date")
			os.Exit(1)
		}
		fmt.Println("begin:", b)
		fmt.Println("end:", e)

		// Get logs, controls parameters
		if len(logStream) == 0 {
			fmt.Fprintln(os.Stderr, "Mandatory option : -l")
			flag.PrintDefaults()
			os.Exit(1)
		}

		// No profile selected
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
			fmt.Fprintf(os.Stderr, "unable to load SDK config: %s", err.Error())
			os.Exit(1)
		}

		// DB file
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			logrus.Errorln("Cannot find HOME environment variable")
			os.Exit(1)
		}
		dbFile := fmt.Sprintf("%s/.ekspodlogs.db", homeDir)

		// Check existence of DB file
		_, err = os.Stat(dbFile)
		if os.IsNotExist(err) {
			logrus.Infoln("DB file not found, create it")
			s, err := sqlite.NewStorage(dbFile)
			if err != nil {
				logrus.Errorln(err.Error())
				os.Exit(1)
			}
			err = s.Init()
			if err != nil {
				logrus.Errorln(err.Error())
				os.Exit(1)
			}
			s.Close()
		}

		s, err := sqlite.NewStorage(dbFile)
		if err != nil {
			logrus.Errorln(err.Error())
			os.Exit(1)
		}
		defer s.Close()

		// Purge DB
		err = s.Purge(ctx)
		if err != nil {
			logrus.Errorln(err.Error())
			os.Exit(1)
		}

		app := app.New(cfg, s)
		if err = app.PrintID(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		if groupName == "" {
			// No groupName specified, try to find it automatically
			groupName, err = app.FindLogGroupAuto(ctx)
			if groupName == "" {
				fmt.Fprintln(os.Stderr, "Log group not found automatically (add option -g)")
				os.Exit(1)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err.Error())
				os.Exit(1)
			}
		}

		err = app.PrintEvents(ctx, groupName, logStream, b.StdTime(), e.StdTime())
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}
