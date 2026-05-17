package termwidth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/termwidth"
)

// DetectFromEnv tests cover the $COLUMNS → default fallback path.
// /dev/tty detection is intentionally excluded (tested manually/in integration).

func TestDetectFromEnv_ValidColumns(t *testing.T) {
	t.Setenv("COLUMNS", "80")

	require.Equal(t, 80, termwidth.DetectFromEnv())
}

func TestDetectFromEnv_InvalidColumns_UsesDefault(t *testing.T) {
	t.Setenv("COLUMNS", "notanumber")

	require.Equal(t, 120, termwidth.DetectFromEnv())
}

func TestDetectFromEnv_ZeroColumns_UsesDefault(t *testing.T) {
	t.Setenv("COLUMNS", "0")

	require.Equal(t, 120, termwidth.DetectFromEnv())
}

func TestDetectFromEnv_NegativeColumns_UsesDefault(t *testing.T) {
	t.Setenv("COLUMNS", "-1")

	require.Equal(t, 120, termwidth.DetectFromEnv())
}

func TestDetectFromEnv_UnsetColumns_UsesDefault(t *testing.T) {
	t.Setenv("COLUMNS", "")

	require.Equal(t, 120, termwidth.DetectFromEnv())
}
