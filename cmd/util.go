package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path"
	"regexp"

	"github.com/google/go-github/github"
	"github.com/kalbasit/swm/ifaces"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	codePkg "github.com/kalbasit/swm/code"
)

var githubClient *github.Client
var code ifaces.Code

func createLogger(cmd *cobra.Command) error {
	// create the logger that pretty prints to the ctx.Writer
	lb := zerolog.New(zerolog.ConsoleWriter{Out: cmd.ErrOrStderr()}).
		With().
		Timestamp()
	if v := viper.GetString("exclude"); v != "" {
		lb = lb.Str("exclude", v)
	}
	if v := viper.GetString("code-path"); v != "" {
		lb = lb.Str("code-path", v)
	}
	if v := viper.GetString("story-name"); v != "" {
		lb = lb.Str("story-name", v)
	}

	if v := viper.GetString("story-branch-name"); v != "" {
		lb = lb.Str("story-branch-name", v)
	}

	log.Logger = lb.Logger().Level(zerolog.InfoLevel)

	if viper.GetBool("debug") {
		log.Logger = log.Logger.Level(zerolog.DebugLevel)
	}

	return nil
}

func createGithubClient() error {
	log.Logger.Debug().Msg("creating a new GitHub client")
	githubAccessToken := viper.GetString("github-access-token")
	if githubAccessToken == "" {
		var err error
		githubTokenPath := path.Join(os.Getenv("HOME"), ".github_token")
		if _, err = os.Stat(githubTokenPath); err == nil {
			var con []byte
			con, err = ioutil.ReadFile(githubTokenPath)
			if err != nil {
				return errors.Wrap(err, "error reading the Github token from ~/.github_token")
			}
			githubAccessToken = string(bytes.TrimSpace(con))
		}
	}
	if githubAccessToken == "" {
		return errors.New("no Github token were provided")
	}

	githubClient = github.NewClient(oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)))

	return nil
}

func createCode() error {
	log.Logger.Debug().Msg("creating a new coder")
	ignorePattern, err := regexp.Compile(viper.GetString("exclude"))
	if err != nil {
		return errors.Wrap(err, "error compiling the exclude pattern")
	}

	code = codePkg.New(githubClient, viper.GetString("code-path"), viper.GetString("story-name"), viper.GetString("story-branch-name"), ignorePattern)

	return code.Scan()
}
