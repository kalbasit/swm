package main

import (
	"errors"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/kalbasit/swm/code"
	"go.i3wm.org/i3"
	cli "gopkg.in/urfave/cli.v2"
)

func getDefaultProfile() string {
	var p string

	// try parsing it from the ACTIVE_PROFILE environment variable.
	p = os.Getenv("ACTIVE_PROFILE")
	// try parsing it from the TMUX environment variable (the session path).
	if p == "" {
		tmuxSocketPath := os.Getenv("TMUX")
		if tmuxSocketPath != "" {
			profileStory := strings.Split(path.Base(tmuxSocketPath), ",")[0]
			profileStoryArr := strings.Split(profileStory, "@")
			if len(profileStoryArr) == 2 {
				p = profileStoryArr[0]
			}
		}
	}
	// finally try parsing it from the i3 workspace
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

	// try parsing it from the ACTIVE_STORY environment variable.
	s = os.Getenv("ACTIVE_STORY")
	// try parsing it from the TMUX environment variable (the session path).
	if s == "" {
		tmuxSocketPath := os.Getenv("TMUX")
		if tmuxSocketPath != "" {
			profileStory := strings.Split(path.Base(tmuxSocketPath), ",")[0]
			profileStoryArr := strings.Split(profileStory, "@")
			if len(profileStoryArr) == 2 {
				s = profileStoryArr[1]
			}
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

func newCoder(ctx *cli.Context) (code.Coder, error) {
	// parse the regex
	ignorePattern, err := regexp.Compile(ctx.String("ignore-pattern"))
	if err != nil {
		return nil, err
	}
	// create a new coder
	return code.New(ctx.String("code-path"), ignorePattern), nil
}
