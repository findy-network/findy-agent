package cmd

import (
	"fmt"

	. "github.com/lainio/err2"
	"github.com/spf13/cobra"
)

var treeDoc = `Prints the findy-agent-cli command structure.

The whole command structure is printed if no argument is given.
If command name is given as argument, only specified command structure is printed.
(Command must be direct subcommand of the root command.)
`

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Prints the findy-agent-cli command structure",
	Long:  treeDoc,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer Return(&err)
		if len(args) == 0 {
			printStructure(rootCmd, "", 0, true)
		} else {
			c, _, e := rootCmd.Find(args)
			Check(e)
			printStructure(c, "", 0, true)
		}
		return nil
	},
}

func printStructure(cmd *cobra.Command, insertion string, level int, last bool) {
	if deepLimit != 0 && level >= deepLimit {
		return
	}
	fmt.Print(insertion)
	if last {
		insertion += " "
		fmt.Print("└── ")
	} else {
		insertion += "│"
		fmt.Print("├── ")
	}
	insertion += "   "
	fmt.Println(cmd.Name())
	for i, subCmd := range cmd.Commands() {
		last := i == len(cmd.Commands())-1
		printStructure(subCmd, insertion, level+1, last)
	}
}

var deepLimit int

func init() {
	treeCmd.PersistentFlags().IntVarP(&deepLimit, "level", "L", 0, "level of the tree, zero is ignored")
	rootCmd.AddCommand(treeCmd)
}
