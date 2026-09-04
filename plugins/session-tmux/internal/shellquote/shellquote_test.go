package shellquote_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/shellquote"
)

// vimCmd stands in for any argument a shell would leave untouched.
const vimCmd = "vim"

func TestArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
		want string
	}{
		{name: "bare word is left alone", arg: vimCmd, want: vimCmd},
		{name: "path is left alone", arg: "/usr/bin/env", want: "/usr/bin/env"},
		{name: "flag is left alone", arg: "--flag=value", want: "--flag=value"},
		{name: "empty string becomes empty quotes", arg: "", want: "''"},
		{name: "space is quoted", arg: "two words", want: "'two words'"},
		{name: "tab is quoted", arg: "a\tb", want: "'a\tb'"},
		{name: "newline is quoted", arg: "a\nb", want: "'a\nb'"},
		{name: "semicolon is quoted", arg: "a;rm -rf /", want: "'a;rm -rf /'"},
		{name: "dollar is quoted", arg: "$HOME", want: "'$HOME'"},
		{name: "backtick is quoted", arg: "`id`", want: "'`id`'"},
		// A leading `#` opens a comment, so an unquoted one makes the shell
		// discard this word and every argument after it -- a silent truncation
		// rather than an error. Reachable from OpenPane argv.
		{name: "leading hash is quoted", arg: "#123", want: "'#123'"},
		{name: "hash mid-word is quoted too", arg: "issue#123", want: "'issue#123'"},
		{name: "glob is quoted", arg: "*.go", want: "'*.go'"},
		{name: "double quote is quoted", arg: `say "hi"`, want: `'say "hi"'`},
		{name: "single quote is escaped", arg: "it's", want: `'it'\''s'`},
		{name: "only a single quote", arg: "'", want: `''\'''`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, shellquote.Arg(tt.arg))
		})
	}
}

func TestArgv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		argv []string
		want string
	}{
		{name: "nil argv is empty", argv: nil, want: ""},
		{name: "single element", argv: []string{vimCmd}, want: vimCmd},
		{
			name: "element with spaces stays one word",
			argv: []string{"my-tool", "--flag", "two words"},
			want: "my-tool --flag 'two words'",
		},
		{
			name: "empty element is preserved",
			argv: []string{"echo", ""},
			want: "echo ''",
		},
		{
			name: "metacharacters cannot escape their element",
			argv: []string{"echo", "a; rm -rf /"},
			want: "echo 'a; rm -rf /'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, shellquote.Argv(tt.argv))
		})
	}
}
