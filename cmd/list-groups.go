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

// listGroupsCmd represents the list-groups command
var listGroupsCmd = &cobra.Command{
	Use:   "list-groups",
	Short: "list-groups lists the log groups",
	Long:  `list-groups lists the log groups. It will list all the log groups in the AWS account.`,
	Run: func(cmd *cobra.Command, args []string) {
		var cfg aws.Config // Configuration to connect to AWS API
		var err error

		ctx := context.Background()
		fmt.Println("profile:", ssoProfile)

		cfg, err = InitAWSConfig(ctx, ssoProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to load SDK config: %s", err.Error())
			os.Exit(1)
		}

		tui := views.NewTerminalView()
		app := app.New(cfg, ssoProfile, nil, tui)
		
		// Configure logger based on debug flag
		logger := NewLoggerWithDebug(debug)
		app.SetLogger(logger)
		
		if err = app.PrintID(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		err = app.ListLogGroups(ctx, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}
