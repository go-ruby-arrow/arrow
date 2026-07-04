// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"testing"

	xipc "github.com/apache/arrow-go/v18/arrow/ipc"
)

var errBoom = errors.New("boom")

// countingWriter tallies the number of Write calls made against it.
type countingWriter struct{ n int }

func (c *countingWriter) Write(p []byte) (int, error) { c.n++; return len(p), nil }

// failingWriter fails exactly on its failOn-th Write call.
type failingWriter struct {
	failOn int
	n      int
}

func (f *failingWriter) Write(p []byte) (int, error) {
	f.n++
	if f.n == f.failOn {
		return 0, errBoom
	}
	return len(p), nil
}

// failingReader fails on its first Read call.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errBoom }

// errAtEOF replays r but substitutes err for the terminal io.EOF, simulating a
// stream that breaks mid-read.
type errAtEOF struct {
	r   io.Reader
	err error
}

func (e *errAtEOF) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == io.EOF {
		return n, e.err
	}
	return n, err
}

func sampleTable(t *testing.T) *Table {
	t.Helper()
	schema := NewSchema(NewField("id", Int64()), NewField("name", StringType()))
	id := mustArray(t, Int64(), []any{int64(1), nil, int64(3)})
	name := mustArray(t, StringType(), []any{"a", "b", "c"})
	tbl, err := NewTable(schema, []*Array{id, name})
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func TestIPCStreamRoundTrip(t *testing.T) {
	tbl := sampleTable(t)
	var buf bytes.Buffer
	if err := WriteTableStream(&buf, tbl); err != nil {
		t.Fatal(err)
	}
	back, err := ReadTableStream(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	assertSameTable(t, tbl, back)
}

func TestIPCFileRoundTrip(t *testing.T) {
	tbl := sampleTable(t)
	var buf bytes.Buffer
	if err := WriteTableFile(&buf, tbl); err != nil {
		t.Fatal(err)
	}
	if !hasArrowMagic(buf.Bytes()) {
		t.Fatal("file output should start with ARROW1 magic")
	}
	back, err := ReadTableFile(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	assertSameTable(t, tbl, back)
}

func assertSameTable(t *testing.T, want, got *Table) {
	t.Helper()
	if got.NumRows() != want.NumRows() || got.NumColumns() != want.NumColumns() {
		t.Fatalf("shape got %dx%d want %dx%d",
			got.NumRows(), got.NumColumns(), want.NumRows(), want.NumColumns())
	}
	gc, _ := got.Column("id")
	if v, _ := gc.Get(0); v != int64(1) {
		t.Errorf("id[0]=%v", v)
	}
	if !gc.NullQ(1) {
		t.Error("id[1] should be null")
	}
	nc, _ := got.Column("name")
	if v, _ := nc.Get(2); v != "c" {
		t.Errorf("name[2]=%v", v)
	}
}

// TestIPCWireCompatReverse: bytes produced by arrow-go's canonical writer read
// back through our reader (the smoke test covers the other direction).
func TestIPCWireCompatReverse(t *testing.T) {
	tbl := sampleTable(t)
	rec := tbl.RecordBatch().Unwrap()
	var buf bytes.Buffer
	w := xipc.NewWriter(&buf, xipc.WithSchema(rec.Schema()))
	if err := w.Write(rec); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	back, err := ReadTableStream(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("our reader rejected canonical bytes: %v", err)
	}
	if back.NumRows() != 3 {
		t.Fatalf("rows %d", back.NumRows())
	}
}

func TestSaveLoad(t *testing.T) {
	tbl := sampleTable(t)
	dir := t.TempDir()
	for _, tc := range []struct {
		name   string
		format Format
	}{
		{"stream.arrows", FormatStream},
		{"file.arrow", FormatFile},
	} {
		path := filepath.Join(dir, tc.name)
		if err := tbl.Save(path, tc.format); err != nil {
			t.Fatalf("Save(%s): %v", tc.name, err)
		}
		back, err := LoadTable(path)
		if err != nil {
			t.Fatalf("LoadTable(%s): %v", tc.name, err)
		}
		assertSameTable(t, tbl, back)
	}
}

func TestSaveErrors(t *testing.T) {
	tbl := sampleTable(t)
	if err := tbl.Save(t.TempDir(), Format(99)); !errors.Is(err, ErrArgument) {
		t.Errorf("bad format err=%v", err)
	}
	// unwritable path (directory that does not exist)
	bad := filepath.Join(t.TempDir(), "nope", "x.arrow")
	if err := tbl.Save(bad, FormatFile); !errors.Is(err, ErrIO) {
		t.Errorf("bad path err=%v", err)
	}
}

func TestLoadErrors(t *testing.T) {
	if _, err := LoadTable(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrIO) {
		t.Errorf("missing file err=%v", err)
	}
}

func TestDecodeTable(t *testing.T) {
	tbl := sampleTable(t)
	// stream path (no magic)
	if _, err := DecodeTable(encodeStream(tbl)); err != nil {
		t.Fatalf("decode stream: %v", err)
	}
	// file path (magic)
	if _, err := DecodeTable(encodeFile(tbl)); err != nil {
		t.Fatalf("decode file: %v", err)
	}
	// short data -> stream path -> error
	if _, err := DecodeTable([]byte{1, 2, 3}); err == nil {
		t.Error("short data should error")
	}
}

func TestHasArrowMagic(t *testing.T) {
	if !hasArrowMagic([]byte("ARROW1xyz")) {
		t.Error("should detect magic")
	}
	if hasArrowMagic([]byte("NOPE12")) {
		t.Error("wrong magic")
	}
	if hasArrowMagic([]byte("AR")) {
		t.Error("too short")
	}
}

func TestWriteStreamErrors(t *testing.T) {
	tbl := sampleTable(t)
	// Write error: fail on the very first byte.
	if err := WriteTableStream(&failingWriter{failOn: 1}, tbl); !errors.Is(err, ErrIO) {
		t.Errorf("stream write err=%v", err)
	}
	// Close error: succeed through Write, fail on the final (EOS) write.
	total := countWrites(t, func(w io.Writer) error { return WriteTableStream(w, tbl) })
	if err := WriteTableStream(&failingWriter{failOn: total}, tbl); !errors.Is(err, ErrIO) {
		t.Errorf("stream close err (failOn=%d)=%v", total, err)
	}
}

func TestWriteFileErrors(t *testing.T) {
	tbl := sampleTable(t)
	if err := WriteTableFile(&failingWriter{failOn: 1}, tbl); !errors.Is(err, ErrIO) {
		t.Errorf("file write err=%v", err)
	}
	total := countWrites(t, func(w io.Writer) error { return WriteTableFile(w, tbl) })
	if err := WriteTableFile(&failingWriter{failOn: total}, tbl); !errors.Is(err, ErrIO) {
		t.Errorf("file close err (failOn=%d)=%v", total, err)
	}
}

func countWrites(t *testing.T, fn func(io.Writer) error) int {
	t.Helper()
	cw := &countingWriter{}
	if err := fn(cw); err != nil {
		t.Fatalf("counting pass failed: %v", err)
	}
	return cw.n
}

func TestReadStreamErrors(t *testing.T) {
	// NewReader failure: garbage instead of a schema message.
	if _, err := ReadTableStream(bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8})); !errors.Is(err, ErrIO) {
		t.Errorf("bad stream header err=%v", err)
	}
	// Mid-stream failure surfaced by rr.Err(): a stream without an EOS marker
	// whose underlying reader breaks.
	tbl := sampleTable(t)
	rec := tbl.RecordBatch().Unwrap()
	var buf bytes.Buffer
	w := xipc.NewWriter(&buf, xipc.WithSchema(rec.Schema()))
	if err := w.Write(rec); err != nil {
		t.Fatal(err)
	}
	// deliberately do not Close: no end-of-stream marker
	r := &errAtEOF{r: bytes.NewReader(buf.Bytes()), err: errBoom}
	if _, err := ReadTableStream(r); !errors.Is(err, ErrIO) {
		t.Errorf("mid-stream err=%v", err)
	}
}

func TestReadFileErrors(t *testing.T) {
	// ReadAll failure.
	if _, err := ReadTableFile(failingReader{}); !errors.Is(err, ErrIO) {
		t.Errorf("readall err=%v", err)
	}
	// NewFileReader failure: bytes that are not a valid Arrow file.
	if _, err := ReadTableFile(bytes.NewReader([]byte("ARROW1 not really a file"))); !errors.Is(err, ErrIO) {
		t.Errorf("bad file err=%v", err)
	}
	// RecordBatch parse failure: a structurally valid file (intact footer, so
	// NewFileReader and NumRecords succeed) whose record-batch message body is
	// corrupted, so RecordBatch(0) fails. The exact byte layout is
	// architecture-dependent (little- vs big-endian metadata), so scan for a
	// single-byte flip in the message region (before the footer) that triggers
	// the record-batch parse path rather than hard-coding an offset.
	data := encodeFile(sampleTable(t))
	found := false
	for off := 8; off < len(data)-120 && !found; off++ {
		c := append([]byte(nil), data...)
		c[off] ^= 0xFF
		if _, err := ReadTableFile(bytes.NewReader(c)); err != nil &&
			bytes.Contains([]byte(err.Error()), []byte("record batch")) {
			found = true
		}
	}
	if !found {
		t.Error("could not trigger a record-batch parse error via corruption")
	}
}
