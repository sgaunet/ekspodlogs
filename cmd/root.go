package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ekspodlogs",
	Short: "",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// rootCmd.CompletionOptions.DisableDefaultCmd = true

	// groupCmd.Flags().IntVar(&gitlabID, "id", 0, "Gitlab Group ID")
	// groupCmd.Flags().BoolVar(&noRecursiveOption, "no-recursive", false, "Do not list tokens of subgroups and projects")
	// rootCmd.AddCommand(groupCmd)

	// projectCmd.Flags().IntVar(&gitlabID, "id", 0, "Gitlab Project ID")
	// rootCmd.AddCommand(projectCmd)

	syncCmd.Flags().StringVarP(&beginDate, "begin", "b", "", "Begin date")
	syncCmd.Flags().StringVarP(&endDate, "end", "e", "", "End date")
	syncCmd.Flags().StringVarP(&groupName, "group", "g", "", "Group name")
	syncCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile")
	syncCmd.Flags().StringVarP(&logStream, "logstream", "l", "", "Log stream")
	rootCmd.AddCommand(syncCmd)

	// purgeCmd.Flags().StringVarP(&beginDate, "begin", "b", "", "Begin date")
	// purgeCmd.Flags().StringVarP(&endDate, "end", "e", "", "End date")
	// purgeCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile")
	rootCmd.AddCommand(purgeCmd)

	reqCmd.Flags().StringVarP(&beginDate, "begin", "b", "", "Begin date")
	reqCmd.Flags().StringVarP(&endDate, "end", "e", "", "End date")
	reqCmd.Flags().StringVarP(&groupName, "group", "g", "", "Group name")
	reqCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile")
	reqCmd.Flags().StringVarP(&logStream, "logstream", "l", "", "Log stream")
	rootCmd.AddCommand(reqCmd)

	listGroupsCmd.Flags().StringVarP(&ssoProfile, "profile", "p", "", "SSO profile")
	rootCmd.AddCommand(listGroupsCmd)

	rootCmd.AddCommand(versionCmd)
}
