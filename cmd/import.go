package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds/agent"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

var importEnvs = map[string]string{
	"wallet-name": "WALLET_NAME",
	"wallet-key":  "WALLET_KEY",
	"file":        "WALLET_FILE",
	"key":         "WALLET_FILE_KEY",
}

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Command for importing wallet",
	Long: `
Command for importing wallet

Example
	findy-agent-cli tools import \
		--wallet-name MyWallet \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp \
		--key walletImportKey \
		--file /path/to/my-import-wallet
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(importEnvs, cmd.Name())
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		err2.Check(impCmd.Validate())
		if !rootFlags.dryRun {
			err2.Try(impCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var impCmd = agent.ImportCmd{}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	flags := importCmd.Flags()
	flags.StringVar(&impCmd.WalletName, "wallet-name", "", flagInfo("wallet name", importCmd.Name(), importEnvs["wallet-name"]))
	flags.StringVar(&impCmd.WalletKey, "wallet-key", "", flagInfo("wallet key", importCmd.Name(), importEnvs["wallet-key"]))
	flags.StringVar(&impCmd.Filename, "file", "", flagInfo("full import file path", importCmd.Name(), importEnvs["file"]))
	flags.StringVar(&impCmd.Key, "key", "", flagInfo("wallet import key", importCmd.Name(), importEnvs["key"]))

	toolsCmd.AddCommand(importCmd)
}
