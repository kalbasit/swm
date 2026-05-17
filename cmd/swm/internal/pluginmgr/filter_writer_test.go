//nolint:testpackage // white-box test for unexported levelFilterWriter
package pluginmgr

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	hclog "github.com/hashicorp/go-hclog"
)

var errWriteFailed = errors.New("write failed")

type errorWriter struct{ err error }

func (e errorWriter) Write(_ []byte) (int, error) { return 0, e.err }

func TestLevelFilterWriter_FiltersJSONDebugAtWarnLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Warn)

	_, err := fw.Write([]byte(`{"@level":"debug","@message":"plugin address","network":"unix"}` + "\n"))
	require.NoError(t, err)
	require.Empty(t, buf.String())
}

func TestLevelFilterWriter_FiltersJSONTraceAtWarnLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Warn)

	_, err := fw.Write([]byte(`{"@level":"trace","@message":"stdio data"}` + "\n"))
	require.NoError(t, err)
	require.Empty(t, buf.String())
}

func TestLevelFilterWriter_PassesJSONWarnAtWarnLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Warn)

	line := `{"@level":"warn","@message":"something went wrong"}` + "\n"
	_, err := fw.Write([]byte(line))
	require.NoError(t, err)
	require.Equal(t, line, buf.String())
}

func TestLevelFilterWriter_PassesJSONErrorAtWarnLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Warn)

	line := `{"@level":"error","@message":"fatal plugin error"}` + "\n"
	_, err := fw.Write([]byte(line))
	require.NoError(t, err)
	require.Equal(t, line, buf.String())
}

func TestLevelFilterWriter_PassesNonJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Warn)

	line := "FAKESTDERR_MARKER: hello from fakestderr\n"
	_, err := fw.Write([]byte(line))
	require.NoError(t, err)
	require.Equal(t, line, buf.String())
}

func TestLevelFilterWriter_SplitWrites(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Warn)

	// go-plugin logStderr writes line bytes then newline in two separate Write calls.
	line := `{"@level":"debug","@message":"plugin address"}`
	_, err := fw.Write([]byte(line))
	require.NoError(t, err)
	require.Empty(t, buf.String()) // incomplete line — not yet processed

	_, err = fw.Write([]byte{'\n'})
	require.NoError(t, err)
	require.Empty(t, buf.String()) // complete line, but filtered
}

func TestLevelFilterWriter_PassesAllAtDebugLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Debug)

	line := `{"@level":"debug","@message":"plugin address"}` + "\n"
	_, err := fw.Write([]byte(line))
	require.NoError(t, err)
	require.Equal(t, line, buf.String())
}

func TestLevelFilterWriter_WriteErrorReturnsLen(t *testing.T) {
	t.Parallel()

	fw := newLevelFilterWriter(errorWriter{err: errWriteFailed}, hclog.Warn)

	p := []byte(`{"@level":"warn","@message":"something"}` + "\n")
	n, err := fw.Write(p)
	require.ErrorIs(t, err, errWriteFailed)
	require.Equal(t, len(p), n)
}

func TestLevelFilterWriter_MultipleLines(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	fw := newLevelFilterWriter(&buf, hclog.Warn)

	input := "" +
		`{"@level":"debug","@message":"filtered"}` + "\n" +
		"plain text line\n" +
		`{"@level":"warn","@message":"kept"}` + "\n" +
		`{"@level":"trace","@message":"also filtered"}` + "\n"

	_, err := fw.Write([]byte(input))
	require.NoError(t, err)

	out := buf.String()
	require.NotContains(t, out, `"@level":"debug"`)
	require.NotContains(t, out, `"@level":"trace"`)
	require.Contains(t, out, "plain text line")
	require.Contains(t, out, `"@level":"warn"`)
}
