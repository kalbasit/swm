package cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var genDocManCmd = &cobra.Command{
	Use:   "man",
	Short: "Generate man page",
	RunE:  genDocManRun,
}

func init() {
	genDocCmd.AddCommand(genDocManCmd)

	genDocManCmd.Flags().String("path", "", "The path to the tree to generate documentation")
	if err := genDocManCmd.MarkFlagRequired("path"); err != nil {
		panic(err)
	}
	if err := genDocManCmd.MarkFlagFilename("path"); err != nil {
		panic(err)
	}
}

func genDocManRun(cmd *cobra.Command, args []string) error {
	p, err := cmd.Flags().GetString("path")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the path flag")
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return errors.Wrap(err, "error creating the doc directory")
	}
	return doc.GenManTree(cmd.Root(), &doc.GenManHeader{
		Title:   "swm",
		Section: "7",
	}, p)
}
