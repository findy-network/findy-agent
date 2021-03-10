package cmd

import (
	"log"
	"os"

	"github.com/findy-network/findy-agent/cmds/agent"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
)

var invitationEnvs = map[string]string{
	"label": "LABEL",
}

// invitationCmd represents the invitation subcommand
var invitationCmd = &cobra.Command{
	Use:   "invitation",
	Short: "Command for creating invitation message for agent",
	Long: `
Command for creating invitation message for agent	

Example
	findy-agent-cli user invitation \
		--wallet-name MyWallet \
		--wallet-key 6cih1cVgRH8...dv67o8QbufxaTHot3Qxp \
		--label invitation_label
	`,
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		return BindEnvs(invitationEnvs, cmd.Name())
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer err2.Return(&err)
		invitateCmd.WalletName = cFlags.WalletName
		invitateCmd.WalletKey = cFlags.WalletKey
		err2.Check(invitateCmd.Validate())
		if !rootFlags.dryRun {
			cmd.SilenceUsage = true
			err2.Try(invitateCmd.Exec(os.Stdout))
		}
		return nil
	},
}

var invitateCmd = agent.InvitationCmd{}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	invitationCmd.Flags().StringVar(&invitateCmd.Name, "label", "", flagInfo("invitation label", invitationCmd.Name(), invitationEnvs["label"]))

	userCmd.AddCommand(invitationCmd)
	serviceCopy := *invitationCmd
	serviceCmd.AddCommand(&serviceCopy)
}
