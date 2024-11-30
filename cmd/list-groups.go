package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/sgaunet/ekspodlogs/internal/app"
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

		// No profile selected
		if len(ssoProfile) == 0 {
			cfg, err = config.LoadDefaultConfig(ctx)
		} else {
			// Try to connect with the SSO profile put in parameter
			cfg, err = config.LoadDefaultConfig(
				ctx,
				config.WithSharedConfigProfile(ssoProfile),
			)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to load SDK config: %s", err.Error())
			os.Exit(1)
		}

		app := app.New(cfg, ssoProfile, nil)
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
