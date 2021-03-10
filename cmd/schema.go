package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/agent/schema"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// schemaCmd represents the schema command
var schCmd = &cobra.Command{
	Use:   "schema",
	Short: "Parent command for operating with schemas",
	Long: `
Parent command for operating with schemas
	`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

var schCreateEnvs = map[string]string{
	"version":    "VERSION",
	"name":       "NAME",
	"attributes": "ATTRIBUTES",
}

// schCreateCmd represents the schema create subcommand
var schCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Command for creating new schema",
	Long: `
Command for creating new schema

Example
	findy-agent-cli sercive schema create \
		--wallet-name TheNewWallet4 \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp \
		--name my_schema_name \
		--attributes ["field1", "field2", "field3"] \
		--version 1.0
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(schCreateEnvs, "SCHEMA")
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		schAttrs = viper.GetStringSlice("attributes")
		sch := &ssi.Schema{
			Name:    schName,
			Version: schVersion,
			Attrs:   schAttrs,
		}
		schemaCmd := schema.CreateCmd{
			Cmd: cmds.Cmd{
				WalletName: cFlags.WalletName,
				WalletKey:  cFlags.WalletKey,
			},
			Schema: sch,
		}
		err2.Check(schemaCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(schemaCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var schReadEnvs = map[string]string{
	"id": "ID",
}

// schReadCmd represents the schema read subcommand
var schReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Command for getting schema by id",
	Long: `
Command for getting schema by id

Example
	findy-agent-cli sercive schema read \
		--wallet-name TheNewWallet4 \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp \
		--id my_schema_id
`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(schReadEnvs, "SCHEMA")

	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)

		schemaCmd := schema.GetCmd{
			Cmd: cmds.Cmd{
				WalletName: cFlags.WalletName,
				WalletKey:  cFlags.WalletKey,
			},
			ID: schID,
		}
		err2.Check(schemaCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(schemaCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var (
	schVersion string
	schName    string
	schAttrs   []string
	schID      string
)

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	serviceCmd.AddCommand(schCmd)
	userCopy := *schCmd

	f := schCreateCmd.Flags()
	f.StringVar(&schVersion, "version", "1.0", flagInfo("schema version", schCmd.Name(), schCreateEnvs["version"]))
	f.StringVar(&schName, "name", "", flagInfo("schema name", schCmd.Name(), schCreateEnvs["name"]))
	f.StringSliceVar(&schAttrs, "attributes", nil, flagInfo("schema attributes", schCmd.Name(), schCreateEnvs["attributes"]))

	r := schReadCmd.Flags()
	r.StringVar(&schID, "id", "", flagInfo("schema ID", schCmd.Name(), schReadEnvs["id"]))

	schCmd.AddCommand(schCreateCmd)
	schCmd.AddCommand(schReadCmd)
	readCopy := *schReadCmd

	userCopy.AddCommand(&readCopy)
	userCmd.AddCommand(&userCopy)
}
