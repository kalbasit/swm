package main

import (
	"context"
	"errors"

	"github.com/google/go-github/github"
	"github.com/kalbasit/swm/code"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
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
		// pull request
		{
			Name:    "pull-request",
			Usage:   "TODO",
			Aliases: []string{"pr"},
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "github.access_token", Usage: "The access token for accessing Github", EnvVars: []string{"GITHUB_ACCESS_TOKEN"}},
			},
			Before: func(ctx *cli.Context) error {
				code.GithubClient = github.NewClient(oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: ctx.String("github.access_token")},
				)))
				return nil
			},
			Subcommands: []*cli.Command{
				// list
				{
					Name:    "list",
					Usage:   "TODO",
					Aliases: []string{"ls"},
					Action:  coderPullRequestList,
				},
			},
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
	if err = c.Scan(); err != nil {
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

func coderPullRequestList(ctx *cli.Context) error {
	return nil
}
