package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates shell completion scripts",
	Long: `
To load completion run following:

bash:
	source <(findy-agent-cli completion bash)

zsh: 
	source <(findy-agent-cli completion zsh)

To configure your shell to load completions for each session add command 
above to your shell configuration script (e.g. .bash_profile/.zshrc).

`,
	ValidArgs: []string{"bash", "zsh"},
	Args:      cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			rootCmd.GenZshCompletion(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
