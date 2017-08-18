package main

import (
	"errors"
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
				Subcommands: []*cli.Command{
					{
						Name:   "switch-client",
						Usage:  "tmux switch client",
						Action: tmuxSwitchClient,
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "profile", Usage: "The profile for the TMUX session", Value: getDefaultProfile()},
							&cli.StringFlag{Name: "story", Usage: "The story for the TMUX session", Value: getDefaultStory()},
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

func getDefaultProfile() string {
	p := os.Getenv("ACTIVE_PROFILE")
	if p == "" {
		i3_workspace, err := getActiveI3WorkspaceName()
		if err == nil && strings.Contains(i3_workspace, "@") {
			p = strings.Split(i3_workspace, "@")[0]
		}
	}

	return p
}

func getDefaultStory() string {
	s := strings.Split(path.Base(os.Getenv("TMUX")), ",")[0]
	if s == "" {
		i3_workspace, err := getActiveI3WorkspaceName()
		if err == nil && strings.Contains(i3_workspace, "@") {
			s = strings.Split(i3_workspace, "@")[1]
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
