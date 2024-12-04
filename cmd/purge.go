package cmd

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// purgeCmd represents the purge command
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge the local database",
	Long:  `Purge the local database. It will remove all the entries in the local database.`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		ctx := context.Background()
		InitDB() // Initialize the database and exit if an error occurs
		defer s.Close()

		if groupName != "" || podName != "" || ssoProfile != "" {
			err = s.PurgeSpecificLogPodLogs(ctx, ssoProfile, groupName, podName)
			if err != nil {
				logrus.Errorln(err.Error())
				os.Exit(1)
			}
			os.Exit(0)
		}
		// Purge DB
		err = s.PurgeAll(ctx)
		if err != nil {
			logrus.Errorln(err.Error())
			os.Exit(1)
		}
	},
}
