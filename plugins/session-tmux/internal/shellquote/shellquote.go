// Package shellquote renders arguments so that a POSIX shell re-parses them
// back into the values they started as.
//
// tmux takes a command as a single string and hands it to a shell, so anything
// session-tmux passes through tmux — a layout entry's command, or the argv of
// a pane opened via Session.OpenPane — has to survive one round of shell word
// splitting and expansion. Quoting here is what keeps an argument containing a
// space from becoming two arguments, and an argument containing `$(...)` from
// becoming whatever that command printed.
package shellquote

import "strings"

// shellMeta lists the characters that make a word unsafe to hand to a shell
//
// `#` is in the set although it is special only at the start of a word: an
// unquoted leading `#` opens a comment, so the shell discards that word and
// every argument after it. Quoting it everywhere costs a pair of quotes on
// words where it was harmless and removes a silent truncation where it was
// not.
// unquoted: word separators, redirections, expansions, globs, and quotes.
const shellMeta = " \t\n&*;<>|'\"()$[]?~`{}!\\#"

// Arg returns arg quoted so a POSIX shell reads it back as exactly arg.
//
// Words with no shell-special characters are returned untouched, which keeps
// the common case readable in logs and in the tmux command line. Everything
// else is wrapped in single quotes, inside which a shell expands nothing. A
// single quote cannot appear inside single quotes, so each one is replaced by
// the four-character sequence quote-backslash-quote-quote: close the quoted
// run, emit an escaped quote, reopen it.
func Arg(arg string) string {
	if arg == "" {
		return "''"
	}

	if !strings.ContainsAny(arg, shellMeta) {
		return arg
	}

	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

// Argv renders an argument vector as a single shell command string, quoting
// each element so the shell splits it back into the same vector.
func Argv(argv []string) string {
	quoted := make([]string, len(argv))
	for i, a := range argv {
		quoted[i] = Arg(a)
	}

	return strings.Join(quoted, " ")
}
