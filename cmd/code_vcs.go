package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var codeVcsCmd = &cobra.Command{
	Use:   "vcs",
	Short: "Interact with repositories available locally in the code directory",
}

func init() {
	codeCmd.AddCommand(codeVcsCmd)

	if err := viper.BindPFlags(codeVcsCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
