package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/adrg/xdg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configPath string
var version string

var errCodePathIsRequired = errors.New("the code path is required")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "swm",
	Short:   "Story-based Workflow Manager",
	Version: version,
	Long:    `swm (Story-based Workflow Manager) is a Tmux session manager specifically designed for Story-based development workflow`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// By default, the RunE functions of all commands print the usage, and
		// prints the error twice.
		// This was discussed in the following issue and cobra is not going to
		// change that behavior anytime soon, and they recommend turning them off
		// instead: https://github.com/spf13/cobra/issues/340
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		if err := createLogger(cmd); err != nil {
			return errors.Wrap(err, "error creating a logger")
		}
		if err := createCode(); err != nil {
			return errors.Wrap(err, "error creating a code")
		}

		return nil
	},
}

func requireCodePath(cmd *cobra.Command, args []string) error {
	if viper.GetString("code-path") == "" {
		return errCodePathIsRequired
	}

	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// initialize the configuration file at ~/.config/swm/config.yaml
	configPath = path.Join(xdg.ConfigHome, "swm", "config.yaml")

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().String("code-path", "", "The path to the code directory")
	if err := viper.BindPFlag("code-path", rootCmd.PersistentFlags().Lookup("code-path")); err != nil {
		panic(err)
	}
	if err := rootCmd.MarkPersistentFlagDirname("code-path"); err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debugging")
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().String("repositories-dirname", "repositories", "The name of the repositories directory, a child directory of the code-path and the parent directory for all repositories")
	if err := viper.BindPFlag("repositories-dirname", rootCmd.PersistentFlags().Lookup("repositories-dirname")); err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().String("stories-dirname", "stories", "The name of the stories directory, a child directory of the code-path and the parent directory for all stories")
	if err := viper.BindPFlag("stories-dirname", rootCmd.PersistentFlags().Lookup("stories-dirname")); err != nil {
		panic(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// prefix all flag names with SWM_. The underscore at the end is added by
	// viper.
	viper.SetEnvPrefix("SWM")

	// replace dashes with underscore when reading the environment looking for a flag
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// read in environment variables that match
	viper.AutomaticEnv()

	// set the configuration file as defined by the flag
	viper.SetConfigFile(configPath)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		log.Fatal(errors.Wrap(err, "error reading the config file"))
	}
}
