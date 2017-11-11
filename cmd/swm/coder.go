package main

import (
	"errors"
	"regexp"

	"github.com/kalbasit/swm/code"
	"github.com/rs/zerolog/log"
	cli "gopkg.in/urfave/cli.v2"
)

var coderCmd = &cli.Command{
	Name: "coder",
	Subcommands: []*cli.Command{
		// add project
		{
			Name:      "add-project",
			Usage:     "TODO",
			Action:    coderAddProject,
			ArgsUsage: "<url>",
		},
	},
}

func coderAddProject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		log.Debug().Msgf("expecting one argument, the URL to clone. Got %d arguments", ctx.Args())
		return errors.New("expecting one argument as url, required")
	}
	// create a new coder
	c, err := newCoder(ctx)
	if err != nil {
		return err
	}
	if err := c.Scan(); err != nil {
		return err
	}
	// get the profile
	profile, err := c.Profile(ctx.String("profile"))
	if err != nil {
		log.Debug().Str("profile", ctx.String("profile")).Msg("profile not found")
		return err
	}
	// get the story
	story := profile.Story(ctx.String("story"))
	// add this project
	return story.AddProject(ctx.Args().First())
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
