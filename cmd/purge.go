package cmd

import (
	"context"
	"fmt"
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

		fmt.Println("TODO: handle purge of the database with the parameter ssoProfile")
		fmt.Println("TODO: handle purge of the database with the parameter startDate")
		fmt.Println("TODO: handle purge of the database with the parameter endDate")
		// Purge DB
		err = s.Purge(ctx)
		if err != nil {
			logrus.Errorln(err.Error())
			os.Exit(1)
		}
	},
}
