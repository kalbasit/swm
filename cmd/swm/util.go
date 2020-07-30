package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"github.com/kalbasit/swm/code"
	"github.com/kalbasit/swm/ifaces"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.i3wm.org/i3"
	"golang.org/x/oauth2"

	cli "github.com/urfave/cli/v2"
)

func createLogger(ctx *cli.Context) error {
	// create the logger that pretty prints to the ctx.Writer
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: ctx.App.Writer}).
		With().
		Timestamp().
		Str("ignore-pattern", ctx.String("ignore-pattern")).
		Str("code-path", ctx.String("code-path")).
		Str("story-name", ctx.String("story-name")).
		Logger().
		Level(zerolog.InfoLevel)
	// handle debug
	if ctx.Bool("debug") {
		log.Logger = log.Logger.Level(zerolog.DebugLevel)
	}

	return nil
}

func sortCommands(subCmds []*cli.Command) {
	for _, subCmd := range subCmds {
		sort.Sort(cli.FlagsByName(subCmd.Flags))
		sort.Sort(cli.CommandsByName(subCmd.Subcommands))
		if len(subCmd.Subcommands) > 0 {
			sortCommands(subCmd.Subcommands)
		}
	}
}

var githubClient *github.Client

func createGithubClient(ctx *cli.Context) error {
	githubAccessToken := ctx.String("github.access_token")
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
	log.Debug().Str("github.access_token", githubAccessToken).Msg("creating the Github client")

	githubClient = github.NewClient(oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)))

	return nil
}

func getDefaultStoryName() string {
	var s string

	// try parsing it from the SWM_STORY_NAME environment variable.
	s = os.Getenv("SWM_STORY_NAME")
	// try parsing it from the TMUX environment variable (the session path).
	if s == "" {
		if tmuxSocketPath := os.Getenv("TMUX"); tmuxSocketPath != "" {
			s = strings.Split(path.Base(tmuxSocketPath), ",")[0]
		}
	}
	// finally try parsing it from the i3 workspace
	if s == "" {
		i3Workspace, err := getActiveI3WorkspaceName()
		if err == nil && strings.Contains(i3Workspace, "@") {
			s = strings.Split(i3Workspace, "@")[1]
		}
	}

	return s
}

func getActiveI3WorkspaceName() (string, error) {
	// get the workspaces
	workspaces, err := i3.GetWorkspaces()
	if err != nil {
		return "", err
	}
	for _, workspace := range workspaces {
		if workspace.Focused {
			return workspace.Name, nil
		}
	}
	return "", errors.New("no active i3 workspace was found")
}

func newCode(ctx *cli.Context) (ifaces.Code, error) {
	// parse the regex
	ignorePattern, err := regexp.Compile(ctx.String("ignore-pattern"))
	if err != nil {
		return nil, err
	}
	// create a new coder
	return code.New(githubClient, ctx.String("code-path"), ctx.String("story-name"), ignorePattern), nil
}
