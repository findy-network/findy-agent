package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds/agent"
	"github.com/findy-network/findy-agent/cmds/onboard"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

var onboardEnvs = map[string]string{
	"export-file": "EXPORT_FILE",
	"export-key":  "EXPORT_KEY",
	"email":       "EMAIL",
	"salt":        "SALT",
}

// onboardCmd represents the onboard subcommand
var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Command for onboarding agent",
	Long: `
Command for onboarding agent.

If --export-file & --export-key flags are set, 
wallet is exported to that location.
	
Example
	findy-agent-cli user onboard \
		--wallet-name TheNewWallet4 \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp	\
		--email myExampleEmail \
		--salt mySalt
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(onboardEnvs, cmd.Name())
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)

		onbCmd.WalletName = cFlags.WalletName
		onbCmd.WalletKey = cFlags.WalletKey
		onbCmd.AgencyAddr = cFlags.URL
		err2.Check(onbCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(onbCmd.Exec(os.Stdout))
		}

		if onbExpCmd.Filename != "" {
			onbExpCmd.WalletName = cFlags.WalletName
			onbExpCmd.WalletKey = cFlags.WalletKey
			err2.Check(onbExpCmd.Validate())
			if !rootFlags.dryRun {
				err2.Try(onbExpCmd.Exec(os.Stdout))
			}
		}
		return nil
	},
}

var onbCmd = onboard.Cmd{}
var onbExpCmd = agent.ExportCmd{}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	flags := onboardCmd.Flags()
	flags.StringVar(&onbExpCmd.Filename, "export-file", "", flagInfo("full export file path", onboardCmd.Name(), onboardEnvs["export-file"]))
	flags.StringVar(&onbExpCmd.ExportKey, "export-key", "", flagInfo("wallet export key", onboardCmd.Name(), onboardEnvs["export-key"]))
	flags.StringVar(&onbCmd.Email, "email", "", flagInfo("onboarding email", onboardCmd.Name(), onboardEnvs["email"]))
	flags.StringVar(&aCmd.Salt, "salt", "", flagInfo("onboarding salt", onboardCmd.Name(), onboardEnvs["salt"]))

	serviceCopy := *onboardCmd
	userCmd.AddCommand(onboardCmd)
	serviceCmd.AddCommand(&serviceCopy)

}
