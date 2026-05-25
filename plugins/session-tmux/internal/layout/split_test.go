package layout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitPercent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		currentFlex   int
		remainingFlex int
		want          int
	}{
		{
			name:          "equal pair gives 50",
			currentFlex:   1,
			remainingFlex: 1,
			want:          50,
		},
		{
			name:          "three equal panes first split",
			currentFlex:   1,
			remainingFlex: 2,
			want:          66, // floor(200/3)
		},
		{
			name:          "three equal panes second split",
			currentFlex:   1,
			remainingFlex: 1,
			want:          50,
		},
		{
			name:          "weighted 2:1 gives 33",
			currentFlex:   2,
			remainingFlex: 1,
			want:          33,
		},
		{
			name:          "weighted 1:2 gives 66",
			currentFlex:   1,
			remainingFlex: 2,
			want:          66, // floor(200/3)
		},
		{
			name:          "extreme ratio clamped to 99",
			currentFlex:   1,
			remainingFlex: 999,
			want:          99,
		},
		{
			name:          "extreme ratio clamped to 1",
			currentFlex:   999,
			remainingFlex: 1,
			want:          1,
		},
		{
			name:          "zero total returns 50",
			currentFlex:   0,
			remainingFlex: 0,
			want:          50,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := splitPercent(tc.currentFlex, tc.remainingFlex)
			require.Equal(t, tc.want, got)
		})
	}
}
