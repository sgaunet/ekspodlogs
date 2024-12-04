package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sgaunet/ekspodlogs/internal/app"
	"github.com/sgaunet/ekspodlogs/pkg/views"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
		fmt.Println("pod:", podName)
		fmt.Println("begin:", beginDate)
		fmt.Println("end:", endDate)

		b, e, err := ConvertTimeToCarbon(beginDate, endDate)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Println("begin:", b)
		fmt.Println("end:", e)

		// Get logs, controls parameters
		// if len(logStream) == 0 {
		// 	fmt.Fprintln(os.Stderr, "Mandatory option : -l")
		// 	flag.PrintDefaults()
		// 	os.Exit(1)
		// }

		cfg, err = InitAWSConfig(ctx, ssoProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to load SDK config: %s", err.Error())
			os.Exit(1)
		}

		InitDB() // Initialize the database and exit if an error occurs
		defer s.Close()

		// Purge DB
		err = s.PurgeSpecificPeriod(ctx, ssoProfile, groupName, podName, b, e)
		if err != nil {
			logrus.Errorln(err.Error())
			defer s.Close()
			os.Exit(1)
		}

		tui := views.NewTerminalView()
		app := app.New(cfg, ssoProfile, s, tui)
		if err = app.PrintID(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			defer s.Close()
			os.Exit(1)
		}

		if groupName == "" {
			// No groupName specified, try to find it automatically
			groupName, err = app.FindLogGroupAuto(ctx)
			if groupName == "" {
				fmt.Fprintln(os.Stderr, "Log group not found automatically (add option -g)")
				defer s.Close()
				os.Exit(1)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err.Error())
				defer s.Close()
				os.Exit(1)
			}
		}

		err = app.PrintEvents(ctx, groupName, podName, b.StdTime(), e.StdTime())
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			defer s.Close()
			os.Exit(1)
		}
	},
}
