package session

import "testing"

func TestSessionName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "dots replaced with bullet",
			key:  "github.com/kalbasit/swm",
			want: "github•com/kalbasit/swm",
		},
		{
			name: "colons replaced with fullwidth colon",
			key:  "host:8080/org/repo",
			want: "host：8080/org/repo",
		},
		{
			name: "slashes preserved",
			key:  "github.com/org/repo",
			want: "github•com/org/repo",
		},
		{
			name: "org-a/utils and org-b/utils differ",
			key:  "github.com/org-a/utils",
			want: "github•com/org-a/utils",
		},
		{
			name: "org-b/utils produces distinct name from org-a/utils",
			key:  "github.com/org-b/utils",
			want: "github•com/org-b/utils",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := sessionName(tc.key)
			if got != tc.want {
				t.Errorf("sessionName(%q) = %q; want %q", tc.key, got, tc.want)
			}
		})
	}
}
