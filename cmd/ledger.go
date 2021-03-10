package cmd

import (
	"github.com/spf13/cobra"
)

// ledgerCmd represents the ledger command
var ledgerCmd = &cobra.Command{
	Use:   "ledger",
	Short: "Parent command for ledger specific actions",
	Long: `
Parent command for ledger specific actions
	`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

func init() {
	rootCmd.AddCommand(ledgerCmd)
}
