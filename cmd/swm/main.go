package main

import (
	"os"
	"path"
	"sort"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/urfave/cli.v2"
)

var version = ""
var app = &cli.App{
	Name:    "swm",
	Version: version,
	Usage:   "swm <command>",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "profile", Usage: "The profile for the TMUX session", Value: getDefaultProfile()},
		&cli.StringFlag{Name: "story", Usage: "The story for the TMUX session", Value: getDefaultStory()},
		&cli.StringFlag{Name: "code-path", Usage: "The absolute path to the code path", Value: path.Join(os.Getenv("HOME"), "code")},
		&cli.StringFlag{Name: "ignore-pattern", Usage: "The Regex pattern to ignore", Value: "^.snapshots$"},
		&cli.BoolFlag{Name: "debug", Usage: "enable debug mode"},
	},
	Before: func(ctx *cli.Context) error {
		// create the logger that pretty prints to the ctx.Writer
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: ctx.App.Writer}).
			With().
			Timestamp().
			Str("ignore-pattern", ctx.String("ignore-pattern")).
			Str("code-path", ctx.String("code-path")).
			Str("profile", ctx.String("profile")).
			Str("story", ctx.String("story")).
			Logger().
			Level(zerolog.InfoLevel)
		// handle debug
		if ctx.Bool("debug") {
			log.Logger = log.Logger.Level(zerolog.DebugLevel)
		}

		return nil
	},
	Authors: []*cli.Author{
		{
			Name:  "Wael Nasreddine",
			Email: "wael.nasreddine@gmail.com",
		},
	},
	Commands: []*cli.Command{
		// coder for code management
		coderCmd,
		// tmux for switch client
		tmuxCmd,
	},
}

func init() {
	// sort the commands/flags
	// TODO: refactor this to walk the tree rather than manually doing so
	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	for _, subCmds := range app.Commands {
		sort.Sort(cli.FlagsByName(subCmds.Flags))
		sort.Sort(cli.CommandsByName(subCmds.Subcommands))
		for _, subCmds := range subCmds.Subcommands {
			sort.Sort(cli.FlagsByName(subCmds.Flags))
			if len(subCmds.Subcommands) > 0 {
				panic("another subcommand level was added, must add another loop")
			}
		}
	}
}

func main() {
	// run the app
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("error occurred")
	}
}
