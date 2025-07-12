package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sgaunet/ekspodlogs/internal/app"
	"github.com/sgaunet/ekspodlogs/pkg/views"
	"github.com/spf13/cobra"
)

// reqCmd represents the req command
var reqCmd = &cobra.Command{
	Use:   "req",
	Short: "requests the local database",
	Long:  `requests the local database`,
	Run: func(cmd *cobra.Command, args []string) {
		var cfg aws.Config // Configuration to connect to AWS API
		var err error
		ctx := context.Background()

		if beginDate == "" || endDate == "" {
			fmt.Fprintln(os.Stderr, "Mandatory options : -b and -e")
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
			}
			os.Exit(1)
		}

		b, e, err := ConvertTimeToCarbon(beginDate, endDate)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

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
		
		// Configure logger based on debug flag
		logger := NewLoggerWithDebug(debug)
		app.SetLogger(logger)
		
		// if err = app.PrintID(); err != nil {
		// 	fmt.Fprintln(os.Stderr, err.Error())
		// 	os.Exit(1)
		// }

		if groupName == "" {
			// No groupName specified, try to find it automatically
			groupName, err = app.FindLogGroupAuto(ctx)
			if groupName == "" {
				fmt.Fprintln(os.Stderr, "Log group not found automatically (add option -g): ", err.Error())
				os.Exit(1)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err.Error())
				os.Exit(1)
			}
		}

		res, err := app.GetEvents(ctx, ssoProfile, groupName, podName, b, e)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		
		if len(res) == 0 {
			fmt.Println("No logs found for the specified criteria")
			return
		}

		if containerName {
			fmt.Println("Event Time\tContainer Name\tLog")
			for _, r := range res {
				fmt.Printf("%s\t%s\t%s\n",
					r.EventTime.Format("2006-01-02 15:04:05"),
					strings.TrimSpace(r.ContainerName),
					strings.TrimSpace(r.Log),
				)
			}
		} else {
			fmt.Println("Event Time\tLog")
			for _, r := range res {
				fmt.Printf("%s\t%s\n",
					r.EventTime.Format("2006-01-02 15:04:05"),
					strings.TrimSpace(r.Log),
				)
			}
		}
	},
}
