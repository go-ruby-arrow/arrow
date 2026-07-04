// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import "testing"

func TestTypeConstructors(t *testing.T) {
	types := []struct {
		dt   *DataType
		name string
	}{
		{Int8(), "int8"},
		{Int16(), "int16"},
		{Int32(), "int32"},
		{Int64(), "int64"},
		{UInt8(), "uint8"},
		{UInt16(), "uint16"},
		{UInt32(), "uint32"},
		{UInt64(), "uint64"},
		{Float32(), "float32"},
		{Float64(), "float64"},
		{Float(), "float64"},
		{Boolean(), "bool"},
		{StringType(), "utf8"},
		{Date(), "date32"},
	}
	for _, tc := range types {
		if tc.dt.Name() != tc.name {
			t.Errorf("Name() = %q, want %q", tc.dt.Name(), tc.name)
		}
		if tc.dt.String() == "" {
			t.Errorf("String() empty for %q", tc.name)
		}
		if tc.dt.Unwrap() == nil {
			t.Errorf("Unwrap() nil for %q", tc.name)
		}
	}
}

func TestTimestampDecimalListStructTypes(t *testing.T) {
	if Timestamp().Name() != "timestamp" {
		t.Errorf("timestamp name %q", Timestamp().Name())
	}
	if Decimal128(10, 2).Name() != "decimal" {
		t.Errorf("decimal name %q", Decimal128(10, 2).Name())
	}
	lst := ListOf(Int64())
	if lst.Name() != "list" {
		t.Errorf("list name %q", lst.Name())
	}
	st := StructOf(NewField("a", Int64()), NewField("b", StringType()))
	if st.Name() != "struct" {
		t.Errorf("struct name %q", st.Name())
	}
}

func TestDataTypeEqual(t *testing.T) {
	if !Int64().EqualQ(Int64()) {
		t.Fatal("Int64 should equal Int64")
	}
	if Int64().EqualQ(Int32()) {
		t.Fatal("Int64 should not equal Int32")
	}
}

func TestField(t *testing.T) {
	f := NewField("x", Int64())
	if f.Name() != "x" {
		t.Errorf("name %q", f.Name())
	}
	if !f.NullableQ() {
		t.Error("NewField should be nullable")
	}
	if !f.DataType().EqualQ(Int64()) {
		t.Error("field data type")
	}
	if f.String() == "" {
		t.Error("field string empty")
	}
	if f.Unwrap().Name != "x" {
		t.Error("unwrap name")
	}
	nn := NewFieldNonNull("y", StringType())
	if nn.NullableQ() {
		t.Error("NewFieldNonNull should not be nullable")
	}
}

func TestSchema(t *testing.T) {
	s := NewSchema(NewField("a", Int64()), NewField("b", StringType()))
	if s.NumFields() != 2 {
		t.Fatalf("num fields %d", s.NumFields())
	}
	if s.Field(0).Name() != "a" {
		t.Errorf("field 0 %q", s.Field(0).Name())
	}
	if s.Unwrap() == nil {
		t.Error("unwrap nil")
	}
	if s.String() == "" {
		t.Error("schema string empty")
	}
	if len(s.Fields()) != 2 {
		t.Errorf("fields len %d", len(s.Fields()))
	}
	if f, ok := s.FieldByName("b"); !ok || f.Name() != "b" {
		t.Errorf("FieldByName b: %v %v", f, ok)
	}
	if _, ok := s.FieldByName("missing"); ok {
		t.Error("FieldByName missing should be false")
	}
}
