//nolint:testpackage // white-box test for unexported hclogLevelFromSlog
package pluginmgr

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	hclog "github.com/hashicorp/go-hclog"
)

func TestHclogLevelFromSlog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		slogLevel slog.Level
		want      hclog.Level
	}{
		{slog.LevelDebug, hclog.Debug},
		{slog.LevelInfo, hclog.Info},
		{slog.LevelWarn, hclog.Warn},
		{slog.LevelError, hclog.Error},
	}

	for _, tc := range tests {
		t.Run(tc.slogLevel.String(), func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: tc.slogLevel}))
			got := hclogLevelFromSlog(context.Background(), logger)
			require.Equal(t, tc.want, got)
		})
	}
}
