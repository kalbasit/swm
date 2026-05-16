// swm-plugin-picker-fzf is the bundled fzf picker plugin for swm.
package main

import (
	"fmt"
	"os"

	sdkpicker "github.com/kalbasit/swm/sdk/go/picker"

	"github.com/kalbasit/swm/plugins/picker-fzf/internal/picker"
)

func main() {
	f, err := picker.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "swm-plugin-picker-fzf: %v\n", err)
		os.Exit(1)
	}

	if err := sdkpicker.Serve(f); err != nil {
		fmt.Fprintf(os.Stderr, "swm-plugin-picker-fzf: serve: %v\n", err)
		os.Exit(1)
	}
}
