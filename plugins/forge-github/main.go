// swm-plugin-forge-github is the GitHub forge plugin for swm.
package main

import (
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
	sdkforge "github.com/kalbasit/swm/sdk/go/forge"

	"github.com/kalbasit/swm/plugins/forge-github/internal/forge"
)

func main() {
	var hostClient pluginv1.HostClient

	if sock := os.Getenv("SWM_HOST_SOCKET"); sock != "" {
		conn, err := grpc.NewClient(
			sock,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "swm-plugin-forge-github: connecting to host socket: %v\n", err)
			os.Exit(1)
		}

		defer conn.Close() //nolint:errcheck // best-effort close on exit

		hostClient = pluginv1.NewHostClient(conn)
	}

	if err := sdkforge.Serve(forge.New(hostClient)); err != nil {
		fmt.Fprintf(os.Stderr, "swm-plugin-forge-github: serve: %v\n", err)
		os.Exit(1)
	}
}
