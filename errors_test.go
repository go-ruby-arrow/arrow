// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"errors"
	"testing"
)

func TestErrorString(t *testing.T) {
	if got := (&Error{Kind: KindType, Msg: "boom"}).Error(); got != "boom" {
		t.Fatalf("msg-only: %q", got)
	}
	cause := errors.New("cause")
	if got := (&Error{Kind: KindIO, Msg: "wrap", Err: cause}).Error(); got != "wrap: cause" {
		t.Fatalf("msg+cause: %q", got)
	}
	if got := (&Error{Kind: KindIO, Err: cause}).Error(); got != "cause" {
		t.Fatalf("cause-only: %q", got)
	}
}

func TestErrorUnwrapAndIs(t *testing.T) {
	cause := errors.New("cause")
	e := wrapError(KindIO, cause, "io failed")
	if !errors.Is(e, cause) {
		t.Fatal("Unwrap should expose the cause")
	}
	if !errors.Is(e, ErrIO) {
		t.Fatal("errors.Is by kind (match) failed")
	}
	if errors.Is(e, ErrType) {
		t.Fatal("errors.Is by kind (mismatch) should be false")
	}
	if e.Is(errors.New("plain")) {
		t.Fatal("Is against non-*Error should be false")
	}
}

func TestErrorRubyClass(t *testing.T) {
	cases := map[ErrorKind]string{
		KindType:           "TypeError",
		KindIndex:          "IndexError",
		KindArgument:       "ArgumentError",
		KindIO:             "Arrow::Error::Io",
		KindNotImplemented: "NotImplementedError",
		KindError:          "Arrow::Error",
	}
	for kind, want := range cases {
		if got := (&Error{Kind: kind}).RubyClass(); got != want {
			t.Errorf("kind %d: got %q want %q", kind, got, want)
		}
	}
}
