// swm-plugin-vcs-git is the bundled git VCS plugin for swm.
package main

import (
	"fmt"
	"os"

	sdkvcs "github.com/kalbasit/swm/sdk/go/vcs"

	"github.com/kalbasit/swm/plugins/vcs-git/internal/vcs"
)

func main() {
	g, err := vcs.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "swm-plugin-vcs-git: %v\n", err)
		os.Exit(1)
	}

	if err := sdkvcs.Serve(g); err != nil {
		fmt.Fprintf(os.Stderr, "swm-plugin-vcs-git: serve: %v\n", err)
		os.Exit(1)
	}
}
