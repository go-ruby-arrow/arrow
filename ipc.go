// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"bytes"
	"io"
	"os"

	"github.com/apache/arrow-go/v18/arrow/ipc"
)

// Format selects the Arrow IPC serialization used by [Table.Save] and produced
// by the byte encoders, mirroring red-arrow's :arrow_streaming vs :arrow_file.
type Format int

const (
	// FormatStream is the Arrow IPC streaming format (a schema message followed
	// by record-batch messages and an end-of-stream marker).
	FormatStream Format = iota
	// FormatFile is the Arrow IPC file format (a.k.a. Feather v2): the stream
	// body framed by "ARROW1" magic and a random-access footer.
	FormatFile
)

// arrowMagic is the 6-byte marker that opens (and closes) an Arrow IPC file.
const arrowMagic = "ARROW1"

// writeStream encodes one record batch as an Arrow IPC stream onto w.
func writeStream(w io.Writer, r *RecordBatch) error {
	sw := ipc.NewWriter(w, ipc.WithSchema(r.rec.Schema()), ipc.WithAllocator(alloc))
	if err := sw.Write(r.rec); err != nil {
		_ = sw.Close()
		return wrapError(KindIO, err, "write Arrow IPC stream")
	}
	if err := sw.Close(); err != nil {
		return wrapError(KindIO, err, "finalize Arrow IPC stream")
	}
	return nil
}

// writeFile encodes one record batch as an Arrow IPC file onto w.
func writeFile(w io.Writer, r *RecordBatch) error {
	// NewFileWriter never returns an error for a valid schema (it only builds
	// the writer struct); construction is deferred to the first Write.
	fw, _ := ipc.NewFileWriter(w, ipc.WithSchema(r.rec.Schema()), ipc.WithAllocator(alloc))
	if err := fw.Write(r.rec); err != nil {
		_ = fw.Close()
		return wrapError(KindIO, err, "write Arrow IPC file")
	}
	if err := fw.Close(); err != nil {
		return wrapError(KindIO, err, "finalize Arrow IPC file")
	}
	return nil
}

// WriteTableStream writes a table to w in the Arrow IPC streaming format
// (Arrow::Table#save with format: :arrow_streaming).
func WriteTableStream(w io.Writer, t *Table) error { return writeStream(w, t.rec) }

// WriteTableFile writes a table to w in the Arrow IPC file / Feather v2 format
// (Arrow::Table#save with format: :arrow).
func WriteTableFile(w io.Writer, t *Table) error { return writeFile(w, t.rec) }

// encodeStream encodes a table to Arrow IPC stream bytes. Encoding to an
// in-memory buffer cannot fail, so no error is returned.
func encodeStream(t *Table) []byte {
	var buf bytes.Buffer
	_ = writeStream(&buf, t.rec)
	return buf.Bytes()
}

// encodeFile encodes a table to Arrow IPC file bytes. Encoding to an in-memory
// buffer cannot fail, so no error is returned.
func encodeFile(t *Table) []byte {
	var buf bytes.Buffer
	_ = writeFile(&buf, t.rec)
	return buf.Bytes()
}

// ReadTableStream reads an Arrow IPC stream from r into a [Table]
// (Arrow::Table.load of a streaming input). Every record batch in the stream is
// concatenated into the returned table.
func ReadTableStream(r io.Reader) (*Table, error) {
	rr, err := ipc.NewReader(r, ipc.WithAllocator(alloc))
	if err != nil {
		return nil, wrapError(KindIO, err, "open Arrow IPC stream")
	}
	defer rr.Release()

	var batches []*RecordBatch
	for rr.Next() {
		rec := rr.RecordBatch()
		rec.Retain()
		batches = append(batches, wrapRecordBatch(rec))
	}
	if err := rr.Err(); err != nil {
		return nil, wrapError(KindIO, err, "read Arrow IPC stream")
	}
	return NewTableFromRecordBatches(batches...)
}

// ReadTableFile reads an Arrow IPC file (Feather v2) from r into a [Table]
// (Arrow::Table.load of a file input).
func ReadTableFile(r io.Reader) (*Table, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, wrapError(KindIO, err, "read Arrow IPC file")
	}
	fr, err := ipc.NewFileReader(bytes.NewReader(data), ipc.WithAllocator(alloc))
	if err != nil {
		return nil, wrapError(KindIO, err, "open Arrow IPC file")
	}
	defer fr.Close()

	n := fr.NumRecords()
	batches := make([]*RecordBatch, 0, n)
	for i := 0; i < n; i++ {
		rec, err := fr.RecordBatch(i)
		if err != nil {
			return nil, wrapError(KindIO, err, "read record batch %d", i)
		}
		rec.Retain()
		batches = append(batches, wrapRecordBatch(rec))
	}
	return NewTableFromRecordBatches(batches...)
}

// DecodeTable reads a table from in-memory Arrow IPC bytes, auto-detecting the
// file format (by its "ARROW1" magic) versus the streaming format.
func DecodeTable(data []byte) (*Table, error) {
	if hasArrowMagic(data) {
		return ReadTableFile(bytes.NewReader(data))
	}
	return ReadTableStream(bytes.NewReader(data))
}

func hasArrowMagic(data []byte) bool {
	return len(data) >= len(arrowMagic) && string(data[:len(arrowMagic)]) == arrowMagic
}

// Save writes a table to path in the given [Format] (Arrow::Table#save).
func (t *Table) Save(path string, format Format) error {
	var data []byte
	switch format {
	case FormatStream:
		data = encodeStream(t)
	case FormatFile:
		data = encodeFile(t)
	default:
		return newError(KindArgument, "unknown save format %d", format)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return wrapError(KindIO, err, "save table to %s", path)
	}
	return nil
}

// LoadTable reads a table from a file at path, auto-detecting the IPC format
// (Arrow::Table.load).
func LoadTable(path string) (*Table, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, wrapError(KindIO, err, "load table from %s", path)
	}
	return DecodeTable(data)
}
