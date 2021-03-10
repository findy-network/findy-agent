package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds/agent"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

var exportEnvs = map[string]string{
	"wallet-name":       "WALLET_NAME",
	"wallet-key":        "WALLET_KEY",
	"file":              "WALLET_FILE",
	"key":               "WALLET_FILE_KEY",
	"legacy-wallet-key": "WALLET_LEGACY_KEY",
}

// exportCmd represents the export subcommand
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Command for exporting wallet",
	Long: `
Command for exporting wallet

Example
	findy-agent-cli tools export \
		--wallet-name MyWallet \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp \
		--key walletExportKey \
		--file path/to/my-export-wallet
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(exportEnvs, cmd.Name())
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		err2.Check(expCmd.Validate())
		if !rootFlags.dryRun {
			err2.Try(expCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var expCmd = agent.ExportCmd{}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	flags := exportCmd.Flags()
	flags.StringVar(&expCmd.WalletName, "wallet-name", "", flagInfo("wallet name", exportCmd.Name(), exportEnvs["wallet-name"]))
	flags.StringVar(&expCmd.WalletKey, "wallet-key", "", flagInfo("wallet key", exportCmd.Name(), exportEnvs["wallet-key"]))
	flags.StringVar(&expCmd.Filename, "file", "", flagInfo("full export file path", exportCmd.Name(), exportEnvs["file"]))
	flags.StringVar(&expCmd.ExportKey, "key", "", flagInfo("wallet export key", exportCmd.Name(), exportEnvs["key"]))
	flags.BoolVar(&expCmd.WalletKeyLegacy, "legacy-wallet-key", false, flagInfo("use old wallet key", exportCmd.Name(), exportEnvs["legacy-wallet-key"]))

	toolsCmd.AddCommand(exportCmd)
}
