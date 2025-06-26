package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sgaunet/ekspodlogs/internal/app"
	"github.com/sgaunet/ekspodlogs/pkg/views"
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

		InitDB() // Initialize the database and exit if an error occurs

		// Set up signal handling for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		
		// Create a context that will be canceled when a signal is received
		ctx, cancel := context.WithCancel(ctx)
		
		// Start a goroutine to handle signals
		go func() {
			sig := <-sigCh
			fmt.Fprintf(os.Stderr, "Received signal %v, shutting down gracefully...\n", sig)
			// Cancel the context to signal all operations to stop
			cancel()
			os.Exit(0)
		}()
		
		// Ensure database is closed when the function returns normally
		defer func() {
			// Cancel the context and signal handler
			cancel()
			// Close database connection
			if err := s.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing database: %v\n", err)
			}
		}()

		cfg, err = InitAWSConfig(ctx, ssoProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to load SDK config: %s", err.Error())
			os.Exit(1)
		}

		tui := views.NewTerminalView()
		app := app.New(cfg, ssoProfile, s, tui)
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

		if beginDate == "" || endDate == "" {
			fmt.Fprintln(os.Stderr, "begin and end dates must be specified")
			os.Exit(1)
		}

		b, e, err := ConvertTimeToCarbon(beginDate, endDate)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		// Purge DB
		err = s.PurgeSpecificPeriod(ctx, ssoProfile, groupName, podName, b, e)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		err = app.PrintEvents(ctx, groupName, podName, b.StdTime(), e.StdTime())
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}
