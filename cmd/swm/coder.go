package main

import (
	"regexp"

	"github.com/kalbasit/swm/code"
	cli "gopkg.in/urfave/cli.v2"
)

func coderAddProject(ctx *cli.Context) error {
	// create a new coder
	c, err := newCoder(ctx)
	if err != nil {
		return err
	}
	// get the profile
	profile, err := c.Profile(ctx.String("profile"))
	if err != nil {
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
