// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"bytes"
	"testing"
	"time"

	xipc "github.com/apache/arrow-go/v18/arrow/ipc"
)

func TestSmokeRoundTrip(t *testing.T) {
	schema := NewSchema(
		NewField("id", Int64()),
		NewField("name", StringType()),
		NewField("score", Float64()),
		NewField("ok", Boolean()),
		NewField("ts", Timestamp()),
	)
	id, _ := NewArrayOf(Int64(), []any{int64(1), int64(2), nil})
	name, _ := NewArrayOf(StringType(), []any{"a", "b", "c"})
	score, _ := NewArrayOf(Float64(), []any{1.5, 2.5, 3.5})
	ok, _ := NewArrayOf(Boolean(), []any{true, false, true})
	ts, _ := NewArrayOf(Timestamp(), []any{time.Unix(1000, 0), time.Unix(2000, 0), time.Unix(3000, 0)})

	tbl, err := NewTable(schema, []*Array{id, name, score, ok, ts})
	if err != nil {
		t.Fatal(err)
	}
	if tbl.NumRows() != 3 || tbl.NumColumns() != 5 {
		t.Fatalf("rows=%d cols=%d", tbl.NumRows(), tbl.NumColumns())
	}

	// Our stream reads back via arrow-go's canonical ipc.Reader.
	var buf bytes.Buffer
	if err := WriteTableStream(&buf, tbl); err != nil {
		t.Fatal(err)
	}
	rr, err := xipc.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("canonical reader rejected our bytes: %v", err)
	}
	got := 0
	for rr.Next() {
		got += int(rr.RecordBatch().NumRows())
	}
	if err := rr.Err(); err != nil {
		t.Fatal(err)
	}
	rr.Release()
	if got != 3 {
		t.Fatalf("canonical reader saw %d rows", got)
	}

	// Round-trip back into our Table.
	back, err := ReadTableStream(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	col, _ := back.Column("id")
	v0, _ := col.Get(0)
	if v0 != int64(1) {
		t.Fatalf("id[0]=%v", v0)
	}
	if !col.NullQ(2) {
		t.Fatal("id[2] should be null")
	}
}
