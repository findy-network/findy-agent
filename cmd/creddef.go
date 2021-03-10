package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/agent/creddef"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

// creddefCmd represents the creddef command
var creddefCmd = &cobra.Command{
	Use:   "creddef",
	Short: "Parent command for operating with Credential definitions",
	Long: `
Parent command for operating with Credential definitions
	`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCmdNeeded(cmd)
	},
}

var credCreateEnvs = map[string]string{
	"tag":       "TAG",
	"schema-id": "SCHEMA_ID",
}

// createCreddefCmd represents the creddef create subcommand
var createCreddefCmd = &cobra.Command{
	Use:   "create",
	Short: "Command for creating new credential definition",
	Long: `
Command for creating new credential definition.

Example
	findy-agent-cli service creddef create \
		--wallet-name TheNewWallet4 \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp \
		--schema-id my_schema_id \
		--tag my_creddef_tag
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(credCreateEnvs, "CREDDEF")
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		credDefCmd := creddef.CreateCmd{
			Cmd: cmds.Cmd{
				WalletName: cFlags.WalletName,
				WalletKey:  cFlags.WalletKey,
			},
			SchemaID: schID,
			Tag:      credDefTag,
		}
		err2.Check(credDefCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(credDefCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var credReadEnvs = map[string]string{
	"id": "ID",
}

// readCreddefCmd represents the creddef read subcommand
var readCreddefCmd = &cobra.Command{
	Use:   "read",
	Short: "Command for getting credential definition by id",
	Long: `
Command for getting credential definition by id

Example
	findy-agent-cli service creddef read \
		--wallet-name TheNewWallet4 \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp	\
		--id my_creddef_id
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(credReadEnvs, "CREDDEF")
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		credDefCmd := creddef.GetCmd{
			Cmd: cmds.Cmd{
				WalletName: cFlags.WalletName,
				WalletKey:  cFlags.WalletKey,
			},
			ID: credDefID,
		}
		err2.Check(credDefCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(credDefCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var (
	credDefTag string
	credDefID  string
)

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	serviceCmd.AddCommand(creddefCmd)
	userCopy := *creddefCmd

	c := createCreddefCmd.Flags()
	c.StringVar(&credDefTag, "tag", "", flagInfo("credential definition tag", creddefCmd.Name(), credCreateEnvs["tag"]))
	c.StringVar(&schID, "schema-id", "", flagInfo("schema ID", creddefCmd.Name(), credCreateEnvs["schema-id"]))

	r := readCreddefCmd.Flags()
	r.StringVar(&credDefID, "id", "", flagInfo("credential definition ID", creddefCmd.Name(), credReadEnvs["id"]))

	creddefCmd.AddCommand(readCreddefCmd)
	readCopy := *readCreddefCmd
	creddefCmd.AddCommand(createCreddefCmd)

	userCopy.AddCommand(&readCopy)
	userCmd.AddCommand(&userCopy)
}
