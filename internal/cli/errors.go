package cli

import "fmt"

// ErrorCode categorises CLI failures so scripts and agents can react without
// parsing human-readable messages. The value is included in the JSON envelope
// and drives the process exit code.
type ErrorCode string

const (
	ErrCodeInternal       ErrorCode = "internal_error"
	ErrCodeUsage          ErrorCode = "usage_error"
	ErrCodeNotInitialised ErrorCode = "not_initialised"
	ErrCodeNotFound       ErrorCode = "not_found"
	ErrCodeInvalidInput   ErrorCode = "invalid_input"
	ErrCodeCancelled      ErrorCode = "cancelled"
)

// Exit codes are stable:
//   0  success
//   1  internal failure
//   2  usage error
//   3  repo not initialised
//   4  record not found
//   5  invalid input (validation)
//   130 cancelled (SIGINT)
const (
	exitSuccess        = 0
	exitInternal       = 1
	exitUsage          = 2
	exitNotInitialised = 3
	exitNotFound       = 4
	exitInvalidInput   = 5
	exitCancelled      = 130
)

// CLIError carries a machine-readable code alongside the message.
type CLIError struct {
	Code    ErrorCode
	Message string
}

func (e *CLIError) Error() string {
	return e.Message
}

func newError(code ErrorCode, format string, args ...any) *CLIError {
	return &CLIError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// codeFor extracts an ErrorCode from an error, defaulting to internal.
func codeFor(err error) ErrorCode {
	if err == nil {
		return ""
	}
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.Code
	}
	return ErrCodeInternal
}

func exitCodeFor(err error) int {
	switch codeFor(err) {
	case "":
		return exitSuccess
	case ErrCodeUsage:
		return exitUsage
	case ErrCodeNotInitialised:
		return exitNotInitialised
	case ErrCodeNotFound:
		return exitNotFound
	case ErrCodeInvalidInput:
		return exitInvalidInput
	case ErrCodeCancelled:
		return exitCancelled
	default:
		return exitInternal
	}
}
