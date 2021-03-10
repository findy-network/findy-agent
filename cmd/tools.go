package cmd

import (
	"github.com/spf13/cobra"
)

// toolsCmd represents the tools command
var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Parent command for all generic findy-agent-cli tools",
	Long: `
Parent command for all generic findy-agent-cli tools
	`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
