package main

import (
	"regexp"

	"github.com/kalbasit/swm/code"
	"github.com/kalbasit/swm/tmux"
	cli "gopkg.in/urfave/cli.v2"
)

func tmuxSwitchClient(ctx *cli.Context) error {
	// parse the regex
	ignorePattern, err := regexp.Compile(ctx.String("ignore-pattern"))
	if err != nil {
		return err
	}
	// create a new coder
	c := code.New(ctx.String("code-path"), ignorePattern)
	// scan the code path
	if err := c.Scan(); err != nil {
		return err
	}
	// create a new TMUX manager
	tmuxManager := tmux.New(&tmux.Options{
		Coder:    c,
		Profile:  ctx.String("profile"),
		Story:    ctx.String("story"),
		KillPane: ctx.Bool("kill-pane"),
	})

	return tmuxManager.SwitchClient()
}
