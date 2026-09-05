package exitcode_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/exitcode"
)

var errPlain = errors.New("something went wrong")

func TestFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "nil error is success",
			err:  nil,
			want: 0,
		},
		{
			name: "plain error defaults to 1",
			err:  errPlain,
			want: 1,
		},
		{
			name: "error carrying a code uses it",
			err:  &exitcode.Error{Code: 3, Err: errPlain},
			want: 3,
		},
		{
			name: "wrapped error still carries its code",
			err:  fmt.Errorf("sending text: %w", &exitcode.Error{Code: 3, Err: errPlain}),
			want: 3,
		},
		{
			name: "doubly wrapped error still carries its code",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("inner: %w", &exitcode.Error{Code: 7, Err: errPlain})),
			want: 7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, exitcode.From(tc.err))
		})
	}
}

func TestErrorMessageAndUnwrap(t *testing.T) {
	t.Parallel()

	err := &exitcode.Error{Code: 3, Err: errPlain}

	// The code is metadata for main, not something a reader should have to
	// scrape out of the message, so the message is the wrapped error's own.
	require.Equal(t, errPlain.Error(), err.Error())
	require.ErrorIs(t, err, errPlain)
}

func TestWrap(t *testing.T) {
	t.Parallel()

	require.NoError(t, exitcode.Wrap(3, nil), "wrapping nil must stay nil so callers can return it unconditionally")

	err := exitcode.Wrap(3, errPlain)
	require.ErrorIs(t, err, errPlain)
	require.Equal(t, 3, exitcode.From(err))
}
