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
	t.Run("typical profile/story", func(t *testing.T) {
		tmx := &tmux{options: &Options{
			Profile: "personal",
			Story:   "STORY-123",
		}}

		assert.Equal(t, "personal@STORY-123", tmx.socketName())
	})

	t.Run("profile with a slash", func(t *testing.T) {
		tmx := &tmux{options: &Options{
			Profile: "personal/a",
			Story:   "STORY-123",
		}}

		assert.Equal(t, "personal_a@STORY-123", tmx.socketName())
	})

	t.Run("story with a slash", func(t *testing.T) {
		tmx := &tmux{options: &Options{
			Profile: "personal",
			Story:   "feature/STORY-123",
		}}

		assert.Equal(t, "personal@feature_STORY-123", tmx.socketName())
	})
}
