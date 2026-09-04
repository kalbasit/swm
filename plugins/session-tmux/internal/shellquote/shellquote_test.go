package shellquote_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/shellquote"
)

func TestArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
		want string
	}{
		{name: "bare word is left alone", arg: "vim", want: "vim"},
		{name: "path is left alone", arg: "/usr/bin/env", want: "/usr/bin/env"},
		{name: "flag is left alone", arg: "--flag=value", want: "--flag=value"},
		{name: "empty string becomes empty quotes", arg: "", want: "''"},
		{name: "space is quoted", arg: "two words", want: "'two words'"},
		{name: "tab is quoted", arg: "a\tb", want: "'a\tb'"},
		{name: "newline is quoted", arg: "a\nb", want: "'a\nb'"},
		{name: "semicolon is quoted", arg: "a;rm -rf /", want: "'a;rm -rf /'"},
		{name: "dollar is quoted", arg: "$HOME", want: "'$HOME'"},
		{name: "backtick is quoted", arg: "`id`", want: "'`id`'"},
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
		{name: "single element", argv: []string{"vim"}, want: "vim"},
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
