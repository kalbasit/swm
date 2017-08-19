package main

import (
	"regexp"

	"github.com/kalbasit/swm/code"
	"github.com/kalbasit/swm/tmux"

	cli "gopkg.in/urfave/cli.v2"
)

func tmuxSwitchClient(ctx *cli.Context) error {
	tmuxManager, err := newTmuxManager(ctx)
	if err != nil {
		return err
	}

	return tmuxManager.SwitchClient(ctx.Bool("kill-pane"))
}

func tmuxVimExit(ctx *cli.Context) error {
	tmuxManager, err := newTmuxManager(ctx)
	if err != nil {
		return err
	}

	return tmuxManager.VimExit()
}

func newTmuxManager(ctx *cli.Context) (tmux.Manager, error) {
	// parse the regex
	ignorePattern, err := regexp.Compile(ctx.String("ignore-pattern"))
	if err != nil {
		return nil, err
	}
	// create a new coder
	c := code.New(ctx.String("code-path"), ignorePattern)
	// scan the code path
	if err := c.Scan(); err != nil {
		return nil, err
	}
	// create a new TMUX manager
	tmuxManager := tmux.New(&tmux.Options{
		Coder:   c,
		Profile: ctx.String("profile"),
		Story:   ctx.String("story"),
	})

	return tmuxManager, nil
}
