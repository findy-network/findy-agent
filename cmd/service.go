package cmd

import (
	"log"

	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

var serviceEnvs = map[string]string{
	"wallet-name": "WALLET_NAME",
	"wallet-key":  "WALLET_KEY",
	"agency-url":  "AGENCY_URL",
}

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Parent command for service client",
	Long: `
Parent command for service agent actions.

This command requires a subcommand so command itself does nothing.
Every service subcommand requires --wallet-name & --wallet-key flags to be specified.
--agency-url flag is agency endpoint base address & it has default value of "http://localhost:8080".

Example
	findy-agent-cli service ping \
		--wallet-name TestWallet \
		--wallet-key 6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp
`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(serviceEnvs, cmd.Name())
	},
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	flags := serviceCmd.PersistentFlags()
	flags.StringVar(&cFlags.WalletName, "wallet-name", "", flagInfo("wallet name", serviceCmd.Name(), serviceEnvs["wallet-name"]))
	flags.StringVar(&cFlags.WalletKey, "wallet-key", "", flagInfo("wallet key", serviceCmd.Name(), serviceEnvs["wallet-key"]))
	flags.StringVar(&cFlags.URL, "agency-url", "http://localhost:8080", flagInfo("endpoint base address", serviceCmd.Name(), serviceEnvs["agency-url"]))

	rootCmd.AddCommand(serviceCmd)
}
