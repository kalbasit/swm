package main

import (
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

func tmuxKillServer(ctx *cli.Context) error {
	tmuxManager, err := newTmuxManager(ctx)
	if err != nil {
		return err
	}

	return tmuxManager.KillServer(ctx.Bool("vim-exit"))
}

func newTmuxManager(ctx *cli.Context) (tmux.Manager, error) {
	// create a new coder
	c, err := newCoder(ctx)
	if err != nil {
		return nil, err
	}
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
