// Package exiterr defines Boardwalk's exit-code taxonomy (ADR 0009) and
// the error type that carries it through every layer.
package exiterr

import (
	"errors"
	"fmt"
)

// Exit codes from ADR 0009. Every failure maps to exactly one.
const (
	CodeOK         = 0
	CodeFailure    = 1 // unclassified (bug, unexpected API/network outcome)
	CodeUsage      = 2 // bad flags, invalid value, malformed reference
	CodeAuth       = 3 // gh missing, unauthenticated, missing scopes
	CodeNotFound   = 4 // project, item, field, or option does not exist
	CodePermission = 5 // authenticated but not allowed on this resource
)

// Category names used in the --json error object (ADR 0009).
const (
	CategoryFailure    = "failure"
	CategoryUsage      = "usage"
	CategoryAuth       = "auth"
	CategoryNotFound   = "not_found"
	CategoryPermission = "permission"
)

// Error is a classified failure with user-facing remediation text.
type Error struct {
	Code        int
	Category    string
	Message     string
	Remediation string
	Err         error
}

func (e *Error) Error() string {
	if e.Remediation != "" {
		return e.Message + "\n  " + e.Remediation
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// New builds a classified error.
func New(code int, message string) *Error {
	return &Error{Code: code, Category: categoryFor(code), Message: message}
}

// WithRemediation sets the smallest useful next step for the user.
func (e *Error) WithRemediation(r string) *Error {
	e.Remediation = r
	return e
}

// Wrap classifies an underlying error without losing it.
func Wrap(code int, message string, err error) *Error {
	return &Error{Code: code, Category: categoryFor(code), Message: message, Err: err}
}

// Usagef, Authf, NotFoundf, Permissionf, Failuref are printf-style helpers.
func Usagef(format string, args ...any) *Error {
	return New(CodeUsage, fmt.Sprintf(format, args...))
}

func Authf(format string, args ...any) *Error {
	return New(CodeAuth, fmt.Sprintf(format, args...))
}

func NotFoundf(format string, args ...any) *Error {
	return New(CodeNotFound, fmt.Sprintf(format, args...))
}

func Permissionf(format string, args ...any) *Error {
	return New(CodePermission, fmt.Sprintf(format, args...))
}

func Failuref(format string, args ...any) *Error {
	return New(CodeFailure, fmt.Sprintf(format, args...))
}

// CodeOf extracts the exit code from any error chain; unclassified
// errors map to CodeFailure.
func CodeOf(err error) int {
	var be *Error
	if errors.As(err, &be) {
		return be.Code
	}
	return CodeFailure
}

// Of extracts the classified error from a chain, or synthesises one.
func Of(err error) *Error {
	var be *Error
	if errors.As(err, &be) {
		return be
	}
	return Wrap(CodeFailure, err.Error(), err)
}

func categoryFor(code int) string {
	switch code {
	case CodeUsage:
		return CategoryUsage
	case CodeAuth:
		return CategoryAuth
	case CodeNotFound:
		return CategoryNotFound
	case CodePermission:
		return CategoryPermission
	default:
		return CategoryFailure
	}
}
