// Package exitcode lets an error carry the process exit status it should
// produce.
//
// Only the main package may terminate the process, so a command that needs a
// status other than the generic failure cannot reach for os.Exit itself. It
// returns an error that reports the status instead, and main asks the error
// what to exit with. This keeps the choice of status next to the condition
// that justifies it, without letting the rest of the CLI end the process.
package exitcode

import "errors"

// Failure is the exit status of any error that does not ask for a specific
// one.
const Failure = 1

// Coder is implemented by errors that name the process exit status they should
// produce.
type Coder interface {
	error
	ExitCode() int
}

// Error attaches an exit status to an error. The status is metadata for main;
// it deliberately does not appear in the message, which stays the wrapped
// error's own so that readers see the condition and not the number.
type Error struct {
	Code int
	Err  error
}

// Error returns the wrapped error's message.
func (e *Error) Error() string { return e.Err.Error() }

// ExitCode returns the process exit status this error asks for.
func (e *Error) ExitCode() int { return e.Code }

// Unwrap exposes the wrapped error so errors.Is and errors.As see through the
// exit status.
func (e *Error) Unwrap() error { return e.Err }

// Wrap attaches code to err. A nil err stays nil so a caller can return the
// result of Wrap unconditionally.
func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}

	return &Error{Code: code, Err: err}
}

// From resolves the process exit status for err: zero when there is no error,
// the status carried by the first error in the chain that names one, and
// Failure for anything else.
func From(err error) int {
	if err == nil {
		return 0
	}

	if coder, ok := errors.AsType[Coder](err); ok {
		return coder.ExitCode()
	}

	return Failure
}
