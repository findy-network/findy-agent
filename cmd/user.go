package cmd

import (
	"log"

	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

var userEnvs = map[string]string{
	"wallet-name": "WALLET_NAME",
	"wallet-key":  "WALLET_KEY",
	"agency-url":  "AGENCY_URL",
}

// userCmd represents the user command
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Parent command for user client",
	Long: `
Parent command for user agent actions.

This command requires a subcommand so command itself does nothing.
Every user subcommand requires --wallet-name & --wallet-key flags to be specified.
--agency-url flag is agency endpoint base address & it has default value of "http://localhost:8080".

Example
	findy-agent-cli user ping \
		--wallet-name TestWallet \
		--wallet-key 6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp
`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(userEnvs, cmd.Name())
	},
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	flags := userCmd.PersistentFlags()
	flags.StringVar(&cFlags.WalletName, "wallet-name", "", flagInfo("wallet name", userCmd.Name(), userEnvs["wallet-name"]))
	flags.StringVar(&cFlags.WalletKey, "wallet-key", "", flagInfo("wallet key", userCmd.Name(), userEnvs["wallet-key"]))
	flags.StringVar(&cFlags.URL, "agency-url", "http://localhost:8080", flagInfo("endpoint base address", userCmd.Name(), userEnvs["agency-url"]))

	rootCmd.AddCommand(userCmd)
}
