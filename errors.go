// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import "fmt"

// ErrorKind identifies which node of red-arrow's exception tree an [Error]
// corresponds to. red-arrow raises a small set of Ruby exception classes; the
// kind records which one so a host (rbgo) can re-raise the faithful class.
type ErrorKind int

const (
	// KindError is the base Arrow::Error (a StandardError in Ruby).
	KindError ErrorKind = iota
	// KindType maps to Ruby's TypeError — a value did not fit the column type.
	KindType
	// KindIndex maps to Ruby's IndexError — an out-of-range row/column.
	KindIndex
	// KindArgument maps to Ruby's ArgumentError — a malformed call.
	KindArgument
	// KindIO maps to Arrow::Error::Io — an IPC read/write failure.
	KindIO
	// KindNotImplemented maps to Ruby's NotImplementedError.
	KindNotImplemented
)

// Error is the pure-Go counterpart of red-arrow's Arrow::Error exception tree.
// It carries the [ErrorKind] (so the exact Ruby class can be reconstructed) and
// an optional wrapped cause, and it participates in errors.Is/As via [Error.Is]
// and [Error.Unwrap].
type Error struct {
	Kind ErrorKind
	Msg  string
	Err  error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		if e.Msg == "" {
			return e.Err.Error()
		}
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

// Unwrap returns the wrapped cause, if any, so errors.Is/As traverse it.
func (e *Error) Unwrap() error { return e.Err }

// Is reports whether target is an [*Error] of the same [ErrorKind], letting
// callers write errors.Is(err, arrow.ErrType) against the sentinel values.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Kind == e.Kind
}

// RubyClass returns the fully-qualified Ruby exception class name a faithful
// host raises for this error, mirroring what red-arrow raises.
func (e *Error) RubyClass() string {
	switch e.Kind {
	case KindType:
		return "TypeError"
	case KindIndex:
		return "IndexError"
	case KindArgument:
		return "ArgumentError"
	case KindIO:
		return "Arrow::Error::Io"
	case KindNotImplemented:
		return "NotImplementedError"
	default:
		return "Arrow::Error"
	}
}

// Sentinel values for errors.Is matching by kind.
var (
	// ErrArrow matches any [*Error] regardless of kind is not provided; use the
	// specific sentinels below. ErrType matches KindType errors, etc.
	ErrType           = &Error{Kind: KindType}
	ErrIndex          = &Error{Kind: KindIndex}
	ErrArgument       = &Error{Kind: KindArgument}
	ErrIO             = &Error{Kind: KindIO}
	ErrNotImplemented = &Error{Kind: KindNotImplemented}
)

func newError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

func wrapError(kind ErrorKind, cause error, format string, args ...any) *Error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...), Err: cause}
}
