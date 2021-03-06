package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/tools/go/vcs"
)

var errNotBoth = errors.New("you cannot specify both --go-import-path and --clone-url")
var errOneRequired = errors.New("you must specify either --go-import-path or --clone-url")

var codeVcsCloneCmd = &cobra.Command{
	Use:     "clone",
	Short:   "Clone a repository",
	PreRunE: requireCodePath,
	RunE:    codeVcsCloneCmdRun,
}

func init() {
	codeVcsCmd.AddCommand(codeVcsCloneCmd)

	codeVcsCloneCmd.Flags().String("go-import-path", "", "Clone a repository by its Go import path")
	codeVcsCloneCmd.Flags().String("clone-url", "", "Clone a repository by its clone URL")
}

func codeVcsCloneCmdRun(cmd *cobra.Command, args []string) error {
	gip, err := cmd.Flags().GetString("go-import-path")
	if err != nil {
		return err
	}

	cu, err := cmd.Flags().GetString("clone-url")
	if err != nil {
		return err
	}

	if gip == "" && cu == "" {
		// re-show the usage for this error
		cmd.SilenceUsage = false
		cmd.SilenceErrors = false
		return errOneRequired
	}

	if gip != "" && cu != "" {
		// re-show the usage for this error
		cmd.SilenceUsage = false
		cmd.SilenceErrors = false
		return errNotBoth
	}

	if gip != "" {
		rr, err := vcs.RepoRootForImportPath(gip, viper.GetBool("debug"))
		if err != nil {
			return errors.Wrap(err, "error computing the repository root for the Golang import path")
		}

		cu = rr.Repo
	}

	return code.Clone(cu)
}
