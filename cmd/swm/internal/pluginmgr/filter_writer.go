package pluginmgr

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	hclog "github.com/hashicorp/go-hclog"
)

// levelFilterWriter wraps an io.Writer and suppresses JSON-format hclog lines
// whose level is below the configured threshold. Non-JSON lines always pass through.
//
// go-plugin's logStderr goroutine unconditionally writes all plugin-process stderr
// bytes to ClientConfig.Stderr before any level check (see client.go:1196), so this
// writer provides the necessary host-side filter: plugin debug/trace noise is dropped
// when swm's effective log level is warn or higher.
type levelFilterWriter struct {
	mu    sync.Mutex
	w     io.Writer
	level hclog.Level
	buf   bytes.Buffer
}

func newLevelFilterWriter(w io.Writer, level hclog.Level) *levelFilterWriter {
	return &levelFilterWriter{w: w, level: level}
}

func (f *levelFilterWriter) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.buf.Write(p) //nolint:wrapcheck // bytes.Buffer.Write never returns an error; ignoring return value is safe

	for {
		data := f.buf.Bytes()

		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}

		line := data[:i]
		f.buf.Next(i + 1)

		if !f.filtered(line) {
			if _, err := f.w.Write(append(line, '\n')); err != nil {
				return len(p), err
			}
		}
	}

	return len(p), nil
}

// filtered reports whether line should be suppressed.
// Only JSON-format hclog entries whose level is below f.level are suppressed.
func (f *levelFilterWriter) filtered(line []byte) bool {
	if len(line) == 0 || line[0] != '{' {
		return false
	}

	var entry struct {
		Level string `json:"@level"`
	}

	if err := json.Unmarshal(line, &entry); err != nil {
		return false
	}

	lineLevel := hclog.LevelFromString(entry.Level)
	if lineLevel == hclog.NoLevel {
		return false
	}

	return lineLevel < f.level
}
