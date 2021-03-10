package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds/steward"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

// stewardCmd represents the steward command
var stewardCmd = &cobra.Command{
	Use:   "steward",
	Short: "Parent command for steward wallet actions",
	Long: `
Parent command for steward wallet actions
	`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

var stewardCreateEnvs = map[string]string{
	"pool-name":   "POOL_NAME",
	"seed":        "SEED",
	"wallet-name": "WALLET_NAME",
	"wallet-key":  "WALLET_KEY",
}

// stewardCreateCmd represents the steward create subcommand
var stewardCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Command for creating steward wallet",
	Long: `	
Command for creating steward wallet
	
Example
	findy-agent-cli ledger steward create \
		--pool-name findy \
		--seed 000000000000000000000000Steward1 \
		--wallet-name sovrin_steward_wallet \
		--wallet-key 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(stewardCreateEnvs, "STEWARD")
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		err2.Check(createStewardCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(createStewardCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var createStewardCmd = steward.CreateCmd{}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	f := stewardCreateCmd.Flags()
	f.StringVar(&createStewardCmd.PoolName, "pool-name", "FINDY_MEM_LEDGER", flagInfo("pool name", stewardCmd.Name(), stewardCreateEnvs["pool-name"]))
	f.StringVar(&createStewardCmd.StewardSeed, "seed", "000000000000000000000000Steward2", flagInfo("steward seed", stewardCmd.Name(), stewardCreateEnvs["seed"]))
	f.StringVar(&createStewardCmd.Cmd.WalletName, "wallet-name", "", flagInfo("name of the steward wallet", stewardCmd.Name(), stewardCreateEnvs["wallet-name"]))
	f.StringVar(&createStewardCmd.Cmd.WalletKey, "wallet-key", "", flagInfo("steward wallet key", stewardCmd.Name(), stewardCreateEnvs["wallet-key"]))

	stewardCmd.AddCommand(stewardCreateCmd)
	ledgerCmd.AddCommand(stewardCmd)
}
