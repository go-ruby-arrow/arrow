// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"errors"
	"testing"
)

func batchWith(t *testing.T, ids []any, names []any) *RecordBatch {
	t.Helper()
	schema := NewSchema(NewField("id", Int64()), NewField("name", StringType()))
	id := mustArray(t, Int64(), ids)
	name := mustArray(t, StringType(), names)
	rb, err := NewRecordBatch(schema, []*Array{id, name})
	if err != nil {
		t.Fatal(err)
	}
	return rb
}

func TestNewTableAndAccessors(t *testing.T) {
	schema := NewSchema(NewField("id", Int64()), NewField("name", StringType()))
	id := mustArray(t, Int64(), []any{int64(1), int64(2)})
	name := mustArray(t, StringType(), []any{"a", "b"})
	tbl, err := NewTable(schema, []*Array{id, name})
	if err != nil {
		t.Fatal(err)
	}
	if tbl.NumRows() != 2 || tbl.NumColumns() != 2 {
		t.Fatalf("rows=%d cols=%d", tbl.NumRows(), tbl.NumColumns())
	}
	if tbl.Schema().NumFields() != 2 {
		t.Error("schema")
	}
	if tbl.String() == "" {
		t.Error("string empty")
	}
	if tbl.RecordBatch().NumRows() != 2 {
		t.Error("record batch")
	}
	col, err := tbl.Column("name")
	if err != nil || col.Length() != 2 {
		t.Fatalf("Column(name): %v", err)
	}
	h := tbl.ToHash()
	if h["id"][1] != int64(2) {
		t.Errorf("to_h %v", h)
	}
	rows := 0
	if err := tbl.EachRecord(func(int, map[string]any) error { rows++; return nil }); err != nil {
		t.Fatal(err)
	}
	if rows != 2 {
		t.Fatalf("each rows %d", rows)
	}
	tbl.Release()
}

func TestNewTableError(t *testing.T) {
	schema := NewSchema(NewField("id", Int64()))
	name := mustArray(t, StringType(), []any{"a"})
	if _, err := NewTable(schema, []*Array{name}); !errors.Is(err, ErrType) {
		t.Errorf("NewTable type mismatch err=%v", err)
	}
}

func TestTableSlice(t *testing.T) {
	tbl, err := NewTableFromRecordBatches(
		batchWith(t, []any{int64(1), int64(2)}, []any{"a", "b"}),
		batchWith(t, []any{int64(3)}, []any{"c"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.NumRows() != 3 {
		t.Fatalf("combined rows %d", tbl.NumRows())
	}
	sl, err := tbl.Slice(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	col, _ := sl.Column(0)
	if v, _ := col.Get(0); v != int64(2) {
		t.Errorf("slice[0]=%v", v)
	}
	if _, err := tbl.Slice(-1, 1); !errors.Is(err, ErrIndex) {
		t.Errorf("bad slice err=%v", err)
	}
}

func TestConcatTables(t *testing.T) {
	a, _ := NewTableFromRecordBatches(batchWith(t, []any{int64(1)}, []any{"a"}))
	b, _ := NewTableFromRecordBatches(batchWith(t, []any{int64(2)}, []any{"b"}))
	c, err := ConcatTables(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if c.NumRows() != 2 {
		t.Fatalf("concat rows %d", c.NumRows())
	}
	if _, err := ConcatTables(); !errors.Is(err, ErrArgument) {
		t.Errorf("empty concat err=%v", err)
	}
}

func TestNewTableFromRecordBatchesErrors(t *testing.T) {
	if _, err := NewTableFromRecordBatches(); !errors.Is(err, ErrArgument) {
		t.Errorf("empty err=%v", err)
	}
	// column-count mismatch
	oneCol := func() *RecordBatch {
		s := NewSchema(NewField("id", Int64()))
		id := mustArray(t, Int64(), []any{int64(1)})
		rb, _ := NewRecordBatch(s, []*Array{id})
		return rb
	}()
	if _, err := NewTableFromRecordBatches(
		batchWith(t, []any{int64(1)}, []any{"a"}), oneCol,
	); !errors.Is(err, ErrArgument) {
		t.Errorf("column-count mismatch err=%v", err)
	}
	// same column count, incompatible column types -> concat fails
	typedA := func() *RecordBatch {
		s := NewSchema(NewField("x", Int64()))
		a := mustArray(t, Int64(), []any{int64(1)})
		rb, _ := NewRecordBatch(s, []*Array{a})
		return rb
	}()
	typedB := func() *RecordBatch {
		s := NewSchema(NewField("x", StringType()))
		a := mustArray(t, StringType(), []any{"s"})
		rb, _ := NewRecordBatch(s, []*Array{a})
		return rb
	}()
	if _, err := NewTableFromRecordBatches(typedA, typedB); err == nil {
		t.Error("expected concat type-mismatch error")
	}
}

func TestConcatArraysDirect(t *testing.T) {
	a := mustArray(t, Int64(), []any{int64(1)})
	b := mustArray(t, Int64(), []any{int64(2)})
	merged, err := concatArrays([]*Array{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Length() != 2 {
		t.Fatalf("merged len %d", merged.Length())
	}
	// empty input is an error surfaced from arrow-go
	if _, err := concatArrays(nil); err == nil {
		t.Error("expected error from empty concat")
	}
}
