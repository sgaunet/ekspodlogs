package cmd

import (
	"context"
	"fmt"
	"os"

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
		fmt.Println("group:", groupName)
		fmt.Println("profile:", ssoProfile)
		fmt.Println("begin:", beginDate)
		fmt.Println("end:", endDate)

		b, e, err := ConvertTimeToCarbon(beginDate, endDate)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Println("begin:", b)
		fmt.Println("end:", e)

		InitDB() // Initialize the database and exit if an error occurs
		defer s.Close()

		cfg, err = InitAWSConfig(ctx, ssoProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to load SDK config: %s", err.Error())
			s.Close()
			os.Exit(1)
		}
		tui := views.NewTerminalView()
		app := app.New(cfg, ssoProfile, s, tui)
		// if err = app.PrintID(); err != nil {
		// 	fmt.Fprintln(os.Stderr, err.Error())
		// 	os.Exit(1)
		// }

		if groupName == "" {
			// No groupName specified, try to find it automatically
			groupName, err = app.FindLogGroupAuto(ctx)
			if groupName == "" {
				fmt.Fprintln(os.Stderr, "Log group not found automatically (add option -g)")
				s.Close()
				os.Exit(1)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err.Error())
				s.Close()
				os.Exit(1)
			}
		}

		res, err := app.GetEvents(ctx, ssoProfile, groupName, b, e)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			s.Close()
			os.Exit(1)
		}
		for _, r := range res {
			fmt.Println(r)
		}
	},
}
