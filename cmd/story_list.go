package cmd

import (
	"fmt"
	"os"

	"github.com/kalbasit/swm/story"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var codeStoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available stories",
	RunE:  codeStoryListRun,
}

func init() {
	codeStoryCmd.AddCommand(codeStoryListCmd)

	codeStoryListCmd.Flags().Bool("name-only", false, "Show only the names of the stories")
}

func codeStoryListRun(cmd *cobra.Command, args []string) error {
	stories, err := story.List()
	if err != nil {
		return errors.Wrap(err, "error listing the stories")
	}

	sno, err := cmd.Flags().GetBool("name-only")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the --name-only flag")
	}

	if sno {
		for _, s := range stories {
			fmt.Println(s.GetName())
		}

		return nil
	}

	// prepare the table writer and write down the PRs
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"Story Name", "Story Branch", "Created at"})
	for _, s := range stories {
		table.Append([]string{s.GetName(), s.GetBranchName(), s.GetCreatedAt().Format("Mon Jan 2 2006 at 15:04")})
	}
	table.Render()

	return nil
}
