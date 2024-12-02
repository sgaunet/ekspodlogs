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

// syncCmd represents the sync command
var listGroupsCmd = &cobra.Command{
	Use:   "list-groups",
	Short: "List groups",
	Long:  `List groups`,
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
