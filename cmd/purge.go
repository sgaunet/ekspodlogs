package cmd

import (
	"context"
	"fmt"
	"os"

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
		defer s.Close()

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
