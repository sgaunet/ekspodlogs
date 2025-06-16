package cmd

import (
	"fmt"
	"os"

	"github.com/sgaunet/ekspodlogs/pkg/storage/sqlite"
	"github.com/spf13/cobra"
)

var (
	DBPath string
	s      *sqlite.Storage

	// Flags
	beginDate  string
	endDate    string
	groupName  string
	ssoProfile string
	podName    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ekspodlogs",
	Short: "Tool to parse logs of applications in an EKS cluster from AWS Cloudwatch",
	Long: `Tool to parse logs of applications in an EKS cluster from AWS Cloudwatch
	
First, you need to configure your AWS credentials with the AWS CLI.
Then, you will have to synchronise the local database with the logs of cloudwatch for a period.

Finally, you will be able to request the logs of a specific logstream for a period.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	syncCmd.Flags().StringVarP(&beginDate, "begin", "b", "", "Begin date")
	syncCmd.Flags().StringVarP(&endDate, "end", "e", "", "End date")
	syncCmd.Flags().StringVarP(&groupName, "group", "g", "", "Group name (not mandatory if there is only one log group : /aws/containerinsights/<Name of your cluster>/application)")
	syncCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile (not mandatory)")
	syncCmd.Flags().StringVarP(&podName, "pod", "n", "", "string that have to match with the pod name")
	rootCmd.AddCommand(syncCmd)

	purgeCmd.Flags().StringVarP(&groupName, "group", "g", "", "Group name (not mandatory if there is only one log group : /aws/containerinsights/<Name of your cluster>/application)")
	purgeCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile (not mandatory)")
	purgeCmd.Flags().StringVarP(&podName, "pod", "n", "", "string that have to match with the pod name")
	rootCmd.AddCommand(purgeCmd)

	reqCmd.Flags().StringVarP(&beginDate, "begin", "b", "", "Begin date")
	reqCmd.Flags().StringVarP(&endDate, "end", "e", "", "End date")
	reqCmd.Flags().StringVarP(&groupName, "group", "g", "", "Group name (not mandatory if there is only one log group : /aws/containerinsights/<Name of your cluster>/application)")
	reqCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile (not mandatory)")
	reqCmd.Flags().StringVarP(&podName, "podname", "n", "", "string that have to match with the pod name")
	rootCmd.AddCommand(reqCmd)

	listGroupsCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile (not mandatory)")
	rootCmd.AddCommand(listGroupsCmd)

	rootCmd.AddCommand(versionCmd)
}
