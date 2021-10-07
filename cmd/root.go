package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds/agency"
	"github.com/lainio/err2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const envPrefix = "FCLI"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Version: utils.Version,
	Use:     "findy-agent",
	Short:   "Findy agent cli tool",
	Long: `
Findy agent cli tool
	`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		agency.ParseLoggingArgs(rootFlags.logging)
		handleViperFlags(cmd)
		aCmd.PreRun()
	},
}

// Execute root
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// To fix errors printed twice removing the cobra generators next
		// see: https://github.com/spf13/cobra/issues/304
		// fmt.Println(err)

		os.Exit(1)
	}
}

// RootCmd returns a current root command which can be used for adding own
// commands in an own repo.
//  	implCmd.AddCommand(listCmd)
// That's a helper function to extend this CLI with own commands and offering
// same base commands as this CLI.
func RootCmd() *cobra.Command {
	return rootCmd
}

// DryRun returns a value of a dry run flag. That's a helper function to extend
// this CLI with own commands and offering same base commands as this CLI.
func DryRun() bool {
	return rootFlags.dryRun
}

// RootFlags are the common flags
type RootFlags struct {
	cfgFile string
	dryRun  bool
	logging string
}

// ClientFlags agent flags
type ClientFlags struct {
	WalletName string
	WalletKey  string
	URL        string
}

var rootFlags = RootFlags{}

var rootEnvs = map[string]string{
	"config":  "CONFIG",
	"logging": "LOGGING",
	"dry-run": "DRY_RUN",
}

func init() {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	cobra.OnInitialize(initConfig)

	flags := rootCmd.PersistentFlags()
	flags.StringVar(&rootFlags.cfgFile, "config", "", flagInfo("configuration file", "", rootEnvs["config"]))
	flags.StringVar(&rootFlags.logging, "logging", "-logtostderr=true -v=2", flagInfo("logging startup arguments", "", rootEnvs["logging"]))
	flags.BoolVarP(&rootFlags.dryRun, "dry-run", "n", false, flagInfo("perform a trial run with no changes made", "", rootEnvs["dry-run"]))

	err2.Check(viper.BindPFlag("logging", flags.Lookup("logging")))
	err2.Check(viper.BindPFlag("dry-run", flags.Lookup("dry-run")))

	err2.Check(BindEnvs(rootEnvs, ""))

}

func initConfig() {
	viper.SetEnvPrefix(envPrefix)
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	readConfigFile()
	readBoundRootFlags()
}

func readBoundRootFlags() {
	rootFlags.logging = viper.GetString("logging")
	rootFlags.dryRun = viper.GetBool("dry-run")
}

func readConfigFile() {
	cfgEnv := os.Getenv(getEnvName("", "config"))
	if rootFlags.cfgFile != "" || cfgEnv != "" {
		printInfo := true
		if rootFlags.cfgFile == "" {
			rootFlags.cfgFile = cfgEnv
			printInfo = false
		}
		viper.SetConfigFile(rootFlags.cfgFile)
		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil && printInfo {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}

// BindEnvs calls viper.BindEnv with envMap and cmdName which can be empty if
// flag is general.
func BindEnvs(envMap map[string]string, cmdName string) (err error) {
	defer err2.Return(&err)
	for flagKey, envName := range envMap {
		finalEnvName := getEnvName(cmdName, envName)
		err2.Check(viper.BindEnv(flagKey, finalEnvName))
	}
	return nil
}

func flagInfo(info, cmdPrefix, envName string) string {
	return info + ", " + getEnvName(cmdPrefix, envName)
}

func getEnvName(cmdName, envName string) string {
	if cmdName == "" {
		return envPrefix + "_" + strings.ToUpper(envName)
	}
	return envPrefix + "_" + strings.ToUpper(cmdName) + "_" + envName
}

func handleViperFlags(cmd *cobra.Command) {
	setRequiredStringFlags(cmd)
	if cmd.HasParent() {
		handleViperFlags(cmd.Parent())
	}
}

func setRequiredStringFlags(cmd *cobra.Command) {
	defer err2.Catch(func(err error) {
		log.Println(err)
	})

	err2.Check(viper.BindPFlags(cmd.LocalFlags()))
	if cmd.PreRunE != nil {
		err2.Check(cmd.PreRunE(cmd, nil))
	}
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if viper.GetString(f.Name) != "" {
			err2.Check(cmd.LocalFlags().Set(f.Name, viper.GetString(f.Name)))
		}
	})
}

// SubCmdNeeded prints the help and error messages because the cmd is abstract.
func SubCmdNeeded(cmd *cobra.Command) {
	fmt.Println("Subcommand needed!")
	_ = cmd.Help()
	os.Exit(1)
}
