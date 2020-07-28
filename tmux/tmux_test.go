package tmux

import (
	"io/ioutil"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	// discard logs
	log.Logger = zerolog.New(ioutil.Discard)
}

func TestSocketName(t *testing.T) {
	t.Run("typical story", func(t *testing.T) {
		tmx := &tmux{options: &Options{
			StoryName: "STORY-123",
		}}

		assert.Equal(t, "swm-STORY-123", tmx.socketName())
	})

	t.Run("story with a slash", func(t *testing.T) {
		tmx := &tmux{options: &Options{
			StoryName: "feature/STORY-123",
		}}

		assert.Equal(t, "swm-feature_STORY-123", tmx.socketName())
	})
}
