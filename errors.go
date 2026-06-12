package snok

import (
	"errors"
	"fmt"
)

// CommandError is the structured failure shape exposed by Snok command runs.
type CommandError struct {
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Details  any    `json:"details,omitempty"`
	ExitCode int    `json:"exitCode"`
}

func (e *CommandError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *CommandError) normalized() *CommandError {
	if e == nil {
		return nil
	}
	copy := *e
	if copy.Kind == "" {
		copy.Kind = "rune/unexpected"
	}
	if copy.Message == "" {
		copy.Message = copy.Kind
	}
	if copy.ExitCode < 1 || copy.ExitCode > 255 {
		copy.ExitCode = 1
	}
	return &copy
}

func parseError(format string, args ...any) *CommandError {
	return &CommandError{
		Kind:     "rune/invalid-arguments",
		Message:  fmt.Sprintf(format, args...),
		ExitCode: 1,
	}
}

func unexpectedError(err error) *CommandError {
	if err == nil {
		return nil
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) {
		return commandErr.normalized()
	}
	return (&CommandError{
		Kind:     "rune/unexpected",
		Message:  err.Error(),
		ExitCode: 1,
	}).normalized()
}

func combineHookError(original *CommandError, hookErr error) *CommandError {
	if hookErr == nil {
		return original.normalized()
	}
	hook := unexpectedError(hookErr)
	if original == nil {
		return hook
	}
	return (&CommandError{
		Kind:    original.normalized().Kind,
		Message: original.normalized().Message,
		Hint:    original.normalized().Hint,
		Details: map[string]any{
			"original": original.normalized(),
			"hook":     hook,
		},
		ExitCode: original.normalized().ExitCode,
	}).normalized()
}
