package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/adrg/xdg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var defaultCfgFile string
var version string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "swm",
	Short:   "Story-based Workflow Manager",
	Version: version,
	Long:    `TODO: describe what's the point of this`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },

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

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, World!")
		fmt.Printf("cfgFile => %s\n", viper.GetString("config"))
		fmt.Printf("codeDir => %s\n", viper.GetString("code-path"))
		fmt.Printf("repositoriesDirname => %s\n", viper.GetString("repositories-dirname"))
		fmt.Printf("storiesDirname => %s\n", viper.GetString("stories-dirname"))
	},
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
	defaultCfgFile = path.Join(xdg.ConfigHome, "swm", "config.yaml")

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().String("config", defaultCfgFile, "The path to the configuration YAML file")
	if err := rootCmd.MarkPersistentFlagFilename("config"); err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().String("code-path", "", "The path to the code directory")
	if err := rootCmd.MarkPersistentFlagDirname("code-path"); err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debugging")

	rootCmd.PersistentFlags().String("repositories-dirname", "repositories", "The name of the repositories directory, a child directory of the code-path and the parent directory for all repositories")

	rootCmd.PersistentFlags().String("stories-dirname", "stories", "The name of the stories directory, a child directory of the code-path and the parent directory for all stories")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(fmt.Sprintf("error binding cobra persistent flags to viper: %s", err))
	}

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
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
	viper.SetConfigFile(viper.GetString("config"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
