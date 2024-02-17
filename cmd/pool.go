package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds/pool"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/spf13/cobra"
)

// poolCmd represents the pool command
var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Parent command for pool commands",
	Long: `
Parent command for pool commands
	`,
	Run: func(cmd *cobra.Command, _ []string) {
		SubCmdNeeded(cmd)
	},
}

var poolCreateEnvs = map[string]string{
	"name":             "NAME",
	"genesis-txn-file": "GENESIS_TXN_FILE",
}

// createPoolCmd represents the pool create subcommand
var createPoolCmd = &cobra.Command{
	Use:   "create",
	Short: "Command for creating creating pool",
	Long: `
Command for creating creating pool

Example
	findy-agent ledger pool create \
		--name findy-pool \
		--genesis-txn-file my-genesis-txn-file
	`,
	PreRunE: func(*cobra.Command, []string) (err error) {
		return BindEnvs(poolCreateEnvs, "POOL")

	},
	RunE: func(_ *cobra.Command, _ []string) (err error) {
		defer err2.Handle(&err)
		Cmd := pool.CreateCmd{
			Name: poolName,
			Txn:  poolGen,
		}
		try.To(Cmd.Validate())
		if !rootFlags.dryRun {
			try.To1(Cmd.Exec(os.Stdout))
		}
		return nil
	},
}

var poolPingEnvs = map[string]string{
	"name": "NAME",
}

// pingPoolCmd represents the pool ping subcommand
var pingPoolCmd = &cobra.Command{
	Use:   "ping",
	Short: "Command for pinging pool",
	Long: `
Command for pinging pool

Example
	findy-agent ledger pool ping \
		--name findy-pool
	`,
	PreRunE: func(_ *cobra.Command, _ []string) (err error) {
		return BindEnvs(poolPingEnvs, "POOL")
	},
	RunE: func(_ *cobra.Command, _ []string) (err error) {
		defer err2.Handle(&err)
		Cmd := pool.PingCmd{
			Name: poolName,
		}
		try.To(Cmd.Validate())
		if !rootFlags.dryRun {
			try.To1(Cmd.Exec(os.Stdout))
		}
		return nil
	},
}

var (
	poolName string
	poolGen  string
)

func init() {
	defer err2.Catch(err2.Err(func(err error) {
		log.Println(err)
	}))

	f := poolCmd.PersistentFlags()
	f.StringVar(&poolName, "name", "", flagInfo("name of the pool", poolCmd.Name(), poolCreateEnvs["name"]))

	c := createPoolCmd.Flags()
	c.StringVar(&poolGen, "genesis-txn-file", "", flagInfo("pool genesis transactions file", poolCmd.Name(), poolCreateEnvs["genesis-txn-file"]))

	ledgerCmd.AddCommand(poolCmd)
	poolCmd.AddCommand(createPoolCmd)
	poolCmd.AddCommand(pingPoolCmd)
}
