package cmd

import (
	"github.com/spf13/cobra"
)

// toolsCmd represents the tools command
var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Parent command for all generic findy-agent tools",
	Long: `
Parent command for all generic findy-agent tools
	`,
	Run: func(cmd *cobra.Command, _ []string) {
		SubCmdNeeded(cmd)
	},
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
