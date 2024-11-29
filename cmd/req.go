package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/dromara/carbon/v2"
	"github.com/sgaunet/ekspodlogs/internal/app"
	"github.com/sgaunet/ekspodlogs/pkg/storage/sqlite"
	"github.com/sirupsen/logrus"
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
		fmt.Println("group:", groupName)
		fmt.Println("profile:", ssoProfile)
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

		app := app.New(cfg, s)
		// if err = app.PrintID(); err != nil {
		// 	fmt.Fprintln(os.Stderr, err.Error())
		// 	os.Exit(1)
		// }

		res, err := app.GetEvents(ctx, groupName, b, e)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		for _, r := range res {
			fmt.Println(r)
		}
	},
}
