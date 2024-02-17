package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds/key"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/spf13/cobra"
)

// keyCmd represents the key subcommand
var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Parent command for handling keys",
	Long: `
Parent command for handling keys
	`,
	Run: func(cmd *cobra.Command, _ []string) {
		SubCmdNeeded(cmd)
	},
}

var keyEnvs = map[string]string{
	"seed": "SEED",
}

// createKeyCmd represents the createkey subcommand
var createKeyCmd = &cobra.Command{
	Use:   "create",
	Short: "Command for creating valid wallet keys",
	Long: `
Command for creating valid wallet keys	

Example	
	findy-agent tools key create \
		--seed 00000000000000000000thisisa_test
	`,
	PreRunE: func(_ *cobra.Command, _ []string) (err error) {
		return BindEnvs(keyEnvs, "KEY")
	},
	RunE: func(_ *cobra.Command, _ []string) (err error) {
		defer err2.Handle(&err)
		try.To(keyCreateCmd.Validate())
		if !rootFlags.dryRun {
			try.To1(keyCreateCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var keyCreateCmd = key.CreateCmd{}

func init() {
	defer err2.Catch(err2.Err(func(err error) {
		log.Println(err)
	}))

	createKeyCmd.Flags().StringVar(&keyCreateCmd.Seed, "seed", "", flagInfo("seed for wallet key creation", keyCmd.Name(), keyEnvs["seed"]))

	toolsCmd.AddCommand(keyCmd)
	keyCmd.AddCommand(createKeyCmd)
}
