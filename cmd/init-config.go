package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Initialize the configuration file",
	RunE:  initConfigRun,
}

func init() {
	rootCmd.AddCommand(initConfigCmd)

	if err := viper.BindPFlags(initConfigCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

// struct to replicate the available configuration items. I tried using
// viper.SafeWriteConfigAs() but it generates a configuration with
// configuration items that should not be stored in the configuration file such
// as the name of the story.
type config struct {
	CodePath            string `yaml:"code-path"`
	Exclude             string `yaml:"exclude"`
	GithubAccessToken   string `yaml:"github-access-token"`
	RepositoriesDirname string `yaml:"repositories-dirname"`
	StoriesDirname      string `yaml:"stories-dirname"`
}

func initConfigRun(cmd *cobra.Command, args []string) error {
	cf := viper.GetString("config")
	if err := os.MkdirAll(path.Dir(cf), 0755); err != nil {
		return errors.Wrap(err, "error creating the parent directory of the config file")
	}

	conf := config{
		CodePath:            viper.GetString("code-path"),
		Exclude:             viper.GetString("exclude"),
		GithubAccessToken:   viper.GetString("github-access-token"),
		RepositoriesDirname: viper.GetString("repositories-dirname"),
		StoriesDirname:      viper.GetString("stories-dirname"),
	}

	bts, err := yaml.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "error converting the config into YAML")
	}

	return ioutil.WriteFile(viper.GetString("config"), bts, 0644)
}
