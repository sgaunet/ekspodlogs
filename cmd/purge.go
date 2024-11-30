package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/sgaunet/ekspodlogs/pkg/storage/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// purgeCmd represents the purge command
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "purge the local database",
	Long:  `purge the local database`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		ctx := context.Background()
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
