package main

import (
	"os"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var version string

var app = &cli.App{
	Name:                 "swm",
	Version:              version,
	Usage:                "swm <command>",
	EnableBashCompletion: true,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "code-path", Usage: "The absolute path to the code path", EnvVars: []string{"CODE_PATH"}},
		&cli.StringFlag{Name: "ignore-pattern", Usage: "The Regex pattern to ignore", Value: "^.snapshots$"},
		&cli.StringFlag{Name: "story-branch-name", Usage: "The name of the branch if different than the name of the story", EnvVars: []string{"SWM_STORY_BRANCH_NAME"}},
		&cli.StringFlag{Name: "story-name", Usage: "The name of the story", EnvVars: []string{"SWM_STORY_NAME"}},
		&cli.BoolFlag{Name: "debug", Usage: "enable debug mode"},
	},
	Before: createLogger,
	Authors: []*cli.Author{
		{
			Name:  "Wael Nasreddine",
			Email: "wael.nasreddine@gmail.com",
		},
	},
	Commands: []*cli.Command{
		// code for code management
		codeCmd,
		// tmux for switch client
		tmuxCmd,
	},
}

func init() {
	// sort the commands/flags
	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	sortCommands(app.Commands)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("error occurred")
	}
}
