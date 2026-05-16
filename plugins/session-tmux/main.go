// swm-plugin-session-tmux is the bundled tmux session plugin for swm.
package main

import (
	"fmt"
	"os"

	sdksession "github.com/kalbasit/swm/sdk/go/session"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/session"
)

func main() {
	t, err := session.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "swm-plugin-session-tmux: %v\n", err)
		os.Exit(1)
	}

	if err := sdksession.Serve(t); err != nil {
		fmt.Fprintf(os.Stderr, "swm-plugin-session-tmux: serve: %v\n", err)
		os.Exit(1)
	}
}
