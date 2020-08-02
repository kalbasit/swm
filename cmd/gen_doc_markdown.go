package cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var genDocMarkdownCmd = &cobra.Command{
	Use:   "markdown",
	Short: "Generate Markdown documentation for the command",
	RunE:  genDocMarkdownRun,
}

func init() {
	genDocCmd.AddCommand(genDocMarkdownCmd)

	genDocMarkdownCmd.Flags().String("path", "", "The path to the tree to generate documentation")
	if err := genDocMarkdownCmd.MarkFlagRequired("path"); err != nil {
		panic(err)
	}
	if err := genDocMarkdownCmd.MarkFlagFilename("path"); err != nil {
		panic(err)
	}
}

func genDocMarkdownRun(cmd *cobra.Command, args []string) error {
	p, err := cmd.Flags().GetString("path")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the path flag")
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return errors.Wrap(err, "error creating the doc directory")
	}
	return doc.GenMarkdownTree(cmd.Root(), p)
}
