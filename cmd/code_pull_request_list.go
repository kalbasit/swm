package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var codePullRequestListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List the pull requests open for this repository over on Github",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := requireCodePath(cmd, args); err != nil {
			return err
		}

		if err := createGithubClient(); err != nil {
			return errors.Wrap(err, "error creating a GitHub client")
		}

		return nil
	},
	RunE: codePullRequestListRun,
}

func init() {
	codePullRequestCmd.AddCommand(codePullRequestListCmd)
}

func codePullRequestListRun(cmd *cobra.Command, args []string) error {
	// get the project from the current PATH
	wd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "error finding the current working directory")
	}
	prj, err := code.GetProjectByAbsolutePath(wd)
	if err != nil {
		return errors.Wrap(err, "error finding the project for the current directory")
	}
	// get the list of prs
	var prs []*github.PullRequest
	prs, err = prj.ListPullRequests(githubClient)
	if err != nil {
		return errors.Wrap(err, "error getting the list of the pull requests")
	}
	if len(prs) == 0 {
		fmt.Println("No pull requests found for the project.")
		return nil
	}
	// prepare the table writer and write down the PRs
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"Number", "Title", "URL", "Created at"})
	for _, pr := range prs {
		table.Append([]string{strconv.Itoa(pr.GetNumber()), pr.GetTitle(), pr.GetHTMLURL(), pr.GetCreatedAt().String()})
	}
	table.Render()

	return nil
}
