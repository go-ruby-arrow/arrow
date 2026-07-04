// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"fmt"

	xarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

// RecordBatch is the pure-Go counterpart of Arrow::RecordBatch — a set of
// equal-length [Array] columns sharing one [Schema]. It is the row-batch unit of
// the Arrow IPC format.
type RecordBatch struct {
	rec xarrow.RecordBatch
}

func wrapRecordBatch(r xarrow.RecordBatch) *RecordBatch { return &RecordBatch{rec: r} }

// Unwrap returns the underlying arrow-go record batch.
func (r *RecordBatch) Unwrap() xarrow.RecordBatch { return r.rec }

// NewRecordBatch builds a record batch from a schema and one [Array] per field
// (Arrow::RecordBatch.new). It returns an [*Error] if the column count, a
// column's type, or a column's length does not match the schema.
func NewRecordBatch(schema *Schema, columns []*Array) (*RecordBatch, error) {
	if len(columns) != schema.NumFields() {
		return nil, newError(KindArgument,
			"expected %d columns for schema, got %d", schema.NumFields(), len(columns))
	}
	cols := make([]xarrow.Array, len(columns))
	nrows := int64(-1)
	for i, c := range columns {
		field := schema.Field(i)
		if !xarrow.TypeEqual(c.arr.DataType(), field.f.Type) {
			return nil, newError(KindType, "column %d (%q) type %s does not match field type %s",
				i, field.Name(), c.arr.DataType(), field.f.Type)
		}
		if nrows < 0 {
			nrows = int64(c.Length())
		} else if int64(c.Length()) != nrows {
			return nil, newError(KindArgument, "column %d (%q) length %d does not match %d",
				i, field.Name(), c.Length(), nrows)
		}
		cols[i] = c.arr
	}
	if nrows < 0 {
		nrows = 0
	}
	return wrapRecordBatch(array.NewRecordBatch(schema.s, cols, nrows)), nil
}

// Schema returns the batch's schema (Arrow::RecordBatch#schema).
func (r *RecordBatch) Schema() *Schema { return wrapSchema(r.rec.Schema()) }

// NumRows returns the number of rows (Arrow::RecordBatch#n_rows).
func (r *RecordBatch) NumRows() int64 { return r.rec.NumRows() }

// NumColumns returns the number of columns (Arrow::RecordBatch#n_columns).
func (r *RecordBatch) NumColumns() int64 { return r.rec.NumCols() }

// Column returns the i-th column, supporting Ruby-style negative indexing. Out
// of range yields an [*Error] of [KindIndex].
func (r *RecordBatch) Column(i int) (*Array, error) {
	n := int(r.rec.NumCols())
	if i < 0 {
		i += n
	}
	if i < 0 || i >= n {
		return nil, newError(KindIndex, "column index %d out of range (%d columns)", i, n)
	}
	return wrapArray(r.rec.Column(i)), nil
}

// ColumnByName returns the column named name and whether it was found.
func (r *RecordBatch) ColumnByName(name string) (*Array, bool) {
	for i := 0; i < int(r.rec.NumCols()); i++ {
		if r.rec.ColumnName(i) == name {
			return wrapArray(r.rec.Column(i)), true
		}
	}
	return nil, false
}

// Get returns a column by integer index or by string name (Arrow::RecordBatch#[]).
func (r *RecordBatch) Get(key any) (*Array, error) {
	switch k := key.(type) {
	case int:
		return r.Column(k)
	case string:
		a, ok := r.ColumnByName(k)
		if !ok {
			return nil, newError(KindIndex, "no column named %q", k)
		}
		return a, nil
	default:
		return nil, newError(KindArgument, "column key must be int or string, got %T", key)
	}
}

// Slice returns a zero-copy row slice [offset, offset+length)
// (Arrow::RecordBatch#slice). Out-of-range bounds yield an [*Error] of
// [KindIndex].
func (r *RecordBatch) Slice(offset, length int64) (*RecordBatch, error) {
	n := r.rec.NumRows()
	end := offset + length
	if offset < 0 || length < 0 || end > n {
		return nil, newError(KindIndex, "slice [%d, %d) out of range (%d rows)", offset, end, n)
	}
	return wrapRecordBatch(r.rec.NewSlice(offset, end)), nil
}

// ToHash returns the batch as an ordered column-name-to-values map
// (Arrow::RecordBatch#to_h). Column order follows the schema.
func (r *RecordBatch) ToHash() map[string][]any {
	out := make(map[string][]any, r.rec.NumCols())
	for i := 0; i < int(r.rec.NumCols()); i++ {
		out[r.rec.ColumnName(i)] = wrapArray(r.rec.Column(i)).ToSlice()
	}
	return out
}

// EachRecord yields each row as a column-name-to-value map
// (Arrow::RecordBatch#each_record). It stops at the first error fn returns.
func (r *RecordBatch) EachRecord(fn func(row int, values map[string]any) error) error {
	n := int(r.rec.NumRows())
	nc := int(r.rec.NumCols())
	for row := 0; row < n; row++ {
		m := make(map[string]any, nc)
		for c := 0; c < nc; c++ {
			m[r.rec.ColumnName(c)] = getValue(r.rec.Column(c), row)
		}
		if err := fn(row, m); err != nil {
			return err
		}
	}
	return nil
}

// String returns a compact human-readable summary of the batch.
func (r *RecordBatch) String() string {
	return fmt.Sprintf("RecordBatch(%d rows, %d columns): %s",
		r.rec.NumRows(), r.rec.NumCols(), r.rec.Schema().String())
}

// Release drops the batch's reference to its buffers.
func (r *RecordBatch) Release() { r.rec.Release() }
