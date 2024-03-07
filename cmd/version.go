package cmd

import (
	"fmt"

	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/spf13/cobra"
)

var versionDoc = ``

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version and build information of the CLI tool",
	Long:  versionDoc,
	RunE: func(_ *cobra.Command, _ []string) (err error) {
		defer err2.Handle(&err)

		try.To1(fmt.Println(utils.Version))
		return nil
	},
}

func init() {
	defer err2.Catch(err2.Err(func(err error) {
		fmt.Println(err)
	}))

	rootCmd.AddCommand(versionCmd)
}
