package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/tools/go/vcs"
)

var errNotBoth = errors.New("you cannot specify both --go-import-path and --clone-url")
var errOneRequired = errors.New("you must specify either --go-import-path or --clone-url")

var codeVcsCloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone a repository",
	RunE:  codeVcsCloneCmdRun,
}

func init() {
	codeVcsCmd.AddCommand(codeVcsCloneCmd)

	codeVcsCloneCmd.Flags().String("go-import-path", "", "Clone a repository by its Go import path")
	codeVcsCloneCmd.Flags().String("clone-url", "", "Clone a repository by its clone URL")

	if err := viper.BindPFlags(codeVcsCloneCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func codeVcsCloneCmdRun(cmd *cobra.Command, args []string) error {
	gip := viper.GetString("go-import-path")
	cu := viper.GetString("clone-url")

	if gip == "" && cu == "" {
		return errOneRequired
	}

	if gip != "" && cu != "" {
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
