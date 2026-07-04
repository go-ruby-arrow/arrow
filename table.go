// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	xarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

// Table is the pure-Go counterpart of Arrow::Table — a schema plus one column
// per field. Unlike Arrow's chunked table, this materializes to a single record
// batch, so [Table] and [RecordBatch] share observable behaviour; the extra
// value a Table carries is multi-batch assembly ([NewTableFromRecordBatches],
// [ConcatTables]) and IPC round-tripping.
type Table struct {
	rec *RecordBatch
}

// NewTable builds a table from a schema and one [Array] per field
// (Arrow::Table.new). Errors mirror [NewRecordBatch].
func NewTable(schema *Schema, columns []*Array) (*Table, error) {
	rb, err := NewRecordBatch(schema, columns)
	if err != nil {
		return nil, err
	}
	return &Table{rec: rb}, nil
}

// NewTableFromRecordBatches assembles one table from a non-empty sequence of
// record batches, concatenating each column across the batches. The batches must
// agree on column count and column types; otherwise an [*Error] is returned.
func NewTableFromRecordBatches(batches ...*RecordBatch) (*Table, error) {
	if len(batches) == 0 {
		return nil, newError(KindArgument, "need at least one record batch to build a table")
	}
	base := batches[0]
	nf := int(base.NumColumns())
	for bi, b := range batches {
		if int(b.NumColumns()) != nf {
			return nil, newError(KindArgument,
				"record batch %d has %d columns, expected %d", bi, b.NumColumns(), nf)
		}
	}
	cols := make([]*Array, nf)
	for c := 0; c < nf; c++ {
		parts := make([]*Array, len(batches))
		for bi, b := range batches {
			col, _ := b.Column(c) // c is in range for every batch
			parts[bi] = col
		}
		merged, err := concatArrays(parts)
		if err != nil {
			return nil, err
		}
		cols[c] = merged
	}
	return NewTable(base.Schema(), cols)
}

// ConcatTables concatenates the rows of two or more tables sharing a schema
// (Arrow::Table#concatenate / #combine).
func ConcatTables(tables ...*Table) (*Table, error) {
	if len(tables) == 0 {
		return nil, newError(KindArgument, "need at least one table to concatenate")
	}
	batches := make([]*RecordBatch, len(tables))
	for i, t := range tables {
		batches[i] = t.rec
	}
	return NewTableFromRecordBatches(batches...)
}

// concatArrays concatenates same-typed arrays into one. It surfaces
// arrow-go's error when the inputs are empty or not identically typed.
func concatArrays(arrays []*Array) (*Array, error) {
	xs := make([]xarrow.Array, len(arrays))
	for i, a := range arrays {
		xs[i] = a.arr
	}
	out, err := array.Concatenate(xs, alloc)
	if err != nil {
		return nil, wrapError(KindError, err, "concatenate arrays")
	}
	return wrapArray(out), nil
}

// Schema returns the table's schema (Arrow::Table#schema).
func (t *Table) Schema() *Schema { return t.rec.Schema() }

// NumRows returns the number of rows (Arrow::Table#n_rows).
func (t *Table) NumRows() int64 { return t.rec.NumRows() }

// NumColumns returns the number of columns (Arrow::Table#n_columns).
func (t *Table) NumColumns() int64 { return t.rec.NumColumns() }

// Column returns a column by integer index or by string name (Arrow::Table#[]).
func (t *Table) Column(key any) (*Array, error) { return t.rec.Get(key) }

// Slice returns the row range [offset, offset+length) as a new table
// (Arrow::Table#slice).
func (t *Table) Slice(offset, length int64) (*Table, error) {
	sl, err := t.rec.Slice(offset, length)
	if err != nil {
		return nil, err
	}
	return &Table{rec: sl}, nil
}

// ToHash returns the table as an ordered column-name-to-values map
// (Arrow::Table#to_h).
func (t *Table) ToHash() map[string][]any { return t.rec.ToHash() }

// EachRecord yields each row as a column-name-to-value map
// (Arrow::Table#each_record).
func (t *Table) EachRecord(fn func(row int, values map[string]any) error) error {
	return t.rec.EachRecord(fn)
}

// RecordBatch returns the table's rows as a single [RecordBatch].
func (t *Table) RecordBatch() *RecordBatch { return t.rec }

// String returns the table's canonical string form (Arrow::Table#to_s).
func (t *Table) String() string { return t.rec.String() }

// Release drops the table's reference to its buffers.
func (t *Table) Release() { t.rec.Release() }
