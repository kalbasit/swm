package main

import (
	"errors"
	"log"
	"os"
	"path"
	"strings"

	"github.com/mdirkse/i3ipc-go"
	"gopkg.in/urfave/cli.v2"
)

var version = ""

func main() {
	app := &cli.App{
		Name:    "swm",
		Version: version,
		Usage:   "swm <command>",
		Authors: []*cli.Author{
			{
				Name:  "Wael Nasreddine",
				Email: "wael.nasreddine@gmail.com",
			},
		},
		Commands: []*cli.Command{
			// tmux for switch client
			{
				Name: "tmux",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "profile", Usage: "The profile for the TMUX session", Value: getDefaultProfile()},
					&cli.StringFlag{Name: "story", Usage: "The story for the TMUX session", Value: getDefaultStory()},
					&cli.StringFlag{Name: "code-path", Usage: "The absolute path to the code path", Value: path.Join(os.Getenv("HOME"), "code")},
					&cli.StringFlag{Name: "ignore-pattern", Usage: "The Regex pattern to ignore", Value: "^.snapshots$"},
				},
				Subcommands: []*cli.Command{
					// switch client switches tmux client
					{
						Name:   "switch-client",
						Usage:  "TODO",
						Action: tmuxSwitchClient,
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "kill-pane", Usage: "kill the TMUX pane after switch client"},
						},
					},
					// vim exit will save/exit any open vim
					{
						Name:   "vim-exit",
						Usage:  "TODO",
						Action: tmuxVimExit,
					},
					// kill-server will kill the server
					{
						Name:   "kill-server",
						Usage:  "TODO",
						Action: tmuxKillServer,
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "vim-exit", Usage: "if vim is found running, kill it"},
						},
					},
				},
			},
		},
	}

	// run the app
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("error running the app: %s", err)
	}
}

func getDefaultProfile() string {
	p := os.Getenv("ACTIVE_PROFILE")
	if p == "" {
		i3Workspace, err := getActiveI3WorkspaceName()
		if err == nil && strings.Contains(i3Workspace, "@") {
			p = strings.Split(i3Workspace, "@")[0]
		}
	}

	return p
}

func getDefaultStory() string {
	var s string

	tmuxSocketPath := os.Getenv("TMUX")
	if tmuxSocketPath != "" {
		s = strings.Split(path.Base(tmuxSocketPath), ",")[0]
	}
	if s == "" {
		i3Workspace, err := getActiveI3WorkspaceName()
		if err == nil && strings.Contains(i3Workspace, "@") {
			s = strings.Split(i3Workspace, "@")[1]
		}
	}

	return s
}

func getActiveI3WorkspaceName() (string, error) {
	// create an i3 IPC socket
	ipcsocket, err := i3ipc.GetIPCSocket()
	if err != nil {
		return "", err
	}
	// get the workspaces
	workspaces, err := ipcsocket.GetWorkspaces()
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
