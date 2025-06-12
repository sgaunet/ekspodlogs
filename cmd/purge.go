package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// purgeCmd represents the purge command
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge the local database",
	Long: `Purge the local database. It will remove all the entries in the local database.
Try option -h to see option in order to to purge only specific logs.`,
	Run: func(cmd *cobra.Command, args []string) {
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
			// Close the database connection
			s.Close()
			os.Exit(0)
		}()
		
		// Ensure database is closed when the function returns normally
		defer func() {
			// Cancel the context and signal handler
			cancel()
			// Close database connection
			s.Close()
		}()

		if groupName != "" || podName != "" || ssoProfile != "" {
			err = s.PurgeSpecificLogPodLogs(ctx, ssoProfile, groupName, podName)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			os.Exit(0)
		}
		// Purge DB
		err = s.PurgeAll(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}
