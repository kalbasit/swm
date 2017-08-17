package main

import (
	"os"
	"path"

	"gopkg.in/urfave/cli.v2"
)

func main() {
	app := &cli.App{
		Name:    "swm",
		Version: "0.0.1",
		Usage:   "swm <command>",
		Authors: []*cli.Author{
			{
				Name:  "Wael Nasreddine",
				Email: "wael.nasreddine@gmail.com",
			},
		},
		Commands: []*cli.Command{
			// server starts the server
			{
				Name:   "serve",
				Usage:  "start the gRPC server",
				Action: serve,
				Flags:  []cli.Flag{},
			},

			// tmux for switch client
			{
				Name: "tmux",
				Subcommands: []*cli.Command{
					{
						Name:   "switch-client",
						Usage:  "tmux switch client",
						Action: tmuxSwitchClient,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "profile", Usage: "The profile for the TMUX session", Value: os.Getenv("ACTIVE_PROFILE")},
							&cli.StringFlag{Name: "story", Usage: "The story for the TMUX session", Value: ""},
							&cli.StringFlag{Name: "socket-path", Usage: "the path to the socket name", Value: os.Getenv("TMUX")},
							&cli.StringFlag{Name: "code-path", Usage: "The absolute path to the code path", Value: path.Join(os.Getenv("HOME"), "code")},
							&cli.StringFlag{Name: "ignore-pattern", Usage: "The Regex pattern to ignore", Value: "^.snapshots$"},
							&cli.BoolFlag{Name: "kill-pane", Usage: "kill the TMUX pane after switch client"},
						},
					},
				},
			},
		},
	}

	// run the app
	app.Run(os.Args)
}
