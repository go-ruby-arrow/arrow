// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"errors"
	"testing"
)

func sampleBatch(t *testing.T) *RecordBatch {
	t.Helper()
	schema := NewSchema(NewField("id", Int64()), NewField("name", StringType()))
	id := mustArray(t, Int64(), []any{int64(1), int64(2), int64(3)})
	name := mustArray(t, StringType(), []any{"a", "b", "c"})
	rb, err := NewRecordBatch(schema, []*Array{id, name})
	if err != nil {
		t.Fatal(err)
	}
	return rb
}

func TestRecordBatchBasics(t *testing.T) {
	rb := sampleBatch(t)
	if rb.NumRows() != 3 || rb.NumColumns() != 2 {
		t.Fatalf("rows=%d cols=%d", rb.NumRows(), rb.NumColumns())
	}
	if rb.Schema().NumFields() != 2 {
		t.Error("schema fields")
	}
	if rb.Unwrap() == nil {
		t.Error("unwrap nil")
	}
	if rb.String() == "" {
		t.Error("string empty")
	}
	col, err := rb.Column(0)
	if err != nil || col.Length() != 3 {
		t.Fatalf("Column(0): %v", err)
	}
	if c, _ := rb.Column(-1); c.DataType().Name() != "utf8" {
		t.Error("negative column index")
	}
	if _, err := rb.Column(9); !errors.Is(err, ErrIndex) {
		t.Errorf("Column(9) err=%v", err)
	}
	if _, ok := rb.ColumnByName("id"); !ok {
		t.Error("ColumnByName id")
	}
	if _, ok := rb.ColumnByName("nope"); ok {
		t.Error("ColumnByName nope should be false")
	}
}

func TestRecordBatchGet(t *testing.T) {
	rb := sampleBatch(t)
	if c, err := rb.Get(1); err != nil || c.DataType().Name() != "utf8" {
		t.Fatalf("Get(1): %v", err)
	}
	if c, err := rb.Get("id"); err != nil || c.DataType().Name() != "int64" {
		t.Fatalf("Get(id): %v", err)
	}
	if _, err := rb.Get("missing"); !errors.Is(err, ErrIndex) {
		t.Errorf("Get(missing) err=%v", err)
	}
	if _, err := rb.Get(1.5); !errors.Is(err, ErrArgument) {
		t.Errorf("Get(float) err=%v", err)
	}
}

func TestRecordBatchSlice(t *testing.T) {
	rb := sampleBatch(t)
	sl, err := rb.Slice(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if sl.NumRows() != 2 {
		t.Fatalf("slice rows %d", sl.NumRows())
	}
	col, _ := sl.Column(0)
	if v, _ := col.Get(0); v != int64(2) {
		t.Errorf("slice[0]=%v", v)
	}
	for _, bad := range [][2]int64{{-1, 1}, {0, -1}, {2, 5}} {
		if _, err := rb.Slice(bad[0], bad[1]); !errors.Is(err, ErrIndex) {
			t.Errorf("Slice(%v) err=%v", bad, err)
		}
	}
}

func TestRecordBatchToHashAndEach(t *testing.T) {
	rb := sampleBatch(t)
	h := rb.ToHash()
	if len(h["id"]) != 3 || h["name"][0] != "a" {
		t.Errorf("to_h=%v", h)
	}
	rows := 0
	if err := rb.EachRecord(func(row int, values map[string]any) error {
		rows++
		if row == 0 && values["name"] != "a" {
			t.Errorf("row0 name=%v", values["name"])
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if rows != 3 {
		t.Fatalf("each rows %d", rows)
	}
	boom := errors.New("boom")
	if err := rb.EachRecord(func(int, map[string]any) error { return boom }); err != boom {
		t.Fatalf("each error propagation %v", err)
	}
	rb.Release()
}

func TestNewRecordBatchErrors(t *testing.T) {
	schema := NewSchema(NewField("id", Int64()), NewField("name", StringType()))
	id := mustArray(t, Int64(), []any{int64(1)})
	name := mustArray(t, StringType(), []any{"a"})

	if _, err := NewRecordBatch(schema, []*Array{id}); !errors.Is(err, ErrArgument) {
		t.Errorf("column count err=%v", err)
	}
	// type mismatch: put a string array where int64 is expected
	if _, err := NewRecordBatch(schema, []*Array{name, name}); !errors.Is(err, ErrType) {
		t.Errorf("type mismatch err=%v", err)
	}
	// length mismatch
	id2 := mustArray(t, Int64(), []any{int64(1), int64(2)})
	if _, err := NewRecordBatch(schema, []*Array{id2, name}); !errors.Is(err, ErrArgument) {
		t.Errorf("length mismatch err=%v", err)
	}
}

func TestNewRecordBatchZeroRows(t *testing.T) {
	schema := NewSchema(NewField("id", Int64()))
	empty := mustArray(t, Int64(), []any{})
	rb, err := NewRecordBatch(schema, []*Array{empty})
	if err != nil {
		t.Fatal(err)
	}
	if rb.NumRows() != 0 {
		t.Fatalf("rows %d", rb.NumRows())
	}

	// Empty schema, no columns: nrows cannot be inferred and defaults to 0.
	rb2, err := NewRecordBatch(NewSchema(), []*Array{})
	if err != nil {
		t.Fatal(err)
	}
	if rb2.NumRows() != 0 || rb2.NumColumns() != 0 {
		t.Fatalf("empty batch rows=%d cols=%d", rb2.NumRows(), rb2.NumColumns())
	}
}
