// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	xarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

func mustArray(t *testing.T, dt *DataType, vals []any) *Array {
	t.Helper()
	a, err := NewArrayOf(dt, vals)
	if err != nil {
		t.Fatalf("NewArrayOf(%s): %v", dt.Name(), err)
	}
	return a
}

func TestArrayBuildAllScalarTypes(t *testing.T) {
	cases := []struct {
		dt   *DataType
		in   []any
		want []any
	}{
		{Boolean(), []any{true, false, nil}, []any{true, false, nil}},
		{Int8(), []any{int8(1), 2, nil}, []any{int8(1), int8(2), nil}},
		{Int16(), []any{int16(1), 2}, []any{int16(1), int16(2)}},
		{Int32(), []any{int32(1), 2}, []any{int32(1), int32(2)}},
		{Int64(), []any{int64(1), 2}, []any{int64(1), int64(2)}},
		{UInt8(), []any{uint8(1), 2}, []any{uint8(1), uint8(2)}},
		{UInt16(), []any{uint16(1), 2}, []any{uint16(1), uint16(2)}},
		{UInt32(), []any{uint32(1), 2}, []any{uint32(1), uint32(2)}},
		{UInt64(), []any{uint64(1), 2}, []any{uint64(1), uint64(2)}},
		{Float32(), []any{float32(1.5), 2}, []any{float32(1.5), float32(2)}},
		{Float64(), []any{1.5, 2}, []any{1.5, float64(2)}},
		{StringType(), []any{"a", "b", nil}, []any{"a", "b", nil}},
	}
	for _, tc := range cases {
		a := mustArray(t, tc.dt, tc.in)
		if got := a.ToSlice(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v want %v", tc.dt.Name(), got, tc.want)
		}
		a.Release()
	}
}

func TestArrayTimestampDateDecimal(t *testing.T) {
	ts := mustArray(t, Timestamp(), []any{time.Unix(5, 0).UTC(), int64(6_000_000), int(7_000_000)})
	got, _ := ts.Get(0)
	if !got.(time.Time).Equal(time.Unix(5, 0).UTC()) {
		t.Errorf("timestamp[0]=%v", got)
	}
	g1, _ := ts.Get(1)
	if g1.(time.Time).Unix() != 6 {
		t.Errorf("timestamp[1]=%v", g1)
	}

	d := mustArray(t, Date(), []any{time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC), int32(3), int(4)})
	gd, _ := d.Get(0)
	if y := gd.(time.Time).Year(); y != 2020 {
		t.Errorf("date year %d", y)
	}

	dec := mustArray(t, Decimal128(10, 2), []any{"1.23", 4.5, float32(6.75)})
	gv, _ := dec.Get(0)
	if gv.(string) != "1.23" {
		t.Errorf("decimal[0]=%v", gv)
	}
	gv1, _ := dec.Get(1)
	if gv1.(string) != "4.50" {
		t.Errorf("decimal[1]=%v", gv1)
	}
}

func TestArrayListAndStruct(t *testing.T) {
	lst := mustArray(t, ListOf(Int64()), []any{
		[]any{int64(1), int64(2)},
		nil,
		[]any{int64(3)},
	})
	v0, _ := lst.Get(0)
	if !reflect.DeepEqual(v0, []any{int64(1), int64(2)}) {
		t.Errorf("list[0]=%v", v0)
	}
	if !lst.NullQ(1) {
		t.Error("list[1] should be null")
	}

	st := mustArray(t, StructOf(NewField("n", Int64()), NewField("s", StringType())), []any{
		map[string]any{"n": int64(7), "s": "hi"},
		map[string]any{"n": int64(8)}, // s missing -> null
	})
	v, _ := st.Get(0)
	m := v.(map[string]any)
	if m["n"] != int64(7) || m["s"] != "hi" {
		t.Errorf("struct[0]=%v", m)
	}
	v1, _ := st.Get(1)
	if v1.(map[string]any)["s"] != nil {
		t.Errorf("struct[1].s should be nil, got %v", v1)
	}
}

func TestArrayAccessors(t *testing.T) {
	a := mustArray(t, Int64(), []any{int64(10), nil, int64(30)})
	if a.Length() != 3 {
		t.Fatalf("length %d", a.Length())
	}
	if a.NNulls() != 1 {
		t.Fatalf("nnulls %d", a.NNulls())
	}
	if a.DataType().Name() != "int64" {
		t.Errorf("datatype %s", a.DataType().Name())
	}
	if a.Unwrap() == nil {
		t.Error("unwrap nil")
	}
	if a.String() == "" {
		t.Error("string empty")
	}
	// negative index
	if v, _ := a.Get(-1); v != int64(30) {
		t.Errorf("Get(-1)=%v", v)
	}
	// null read
	if v, _ := a.Get(1); v != nil {
		t.Errorf("Get(1)=%v", v)
	}
	// out of range
	if _, err := a.Get(5); !errors.Is(err, ErrIndex) {
		t.Errorf("Get(5) err=%v", err)
	}
	if _, err := a.Get(-9); !errors.Is(err, ErrIndex) {
		t.Errorf("Get(-9) err=%v", err)
	}
	// NullQ / ValidQ variants
	if !a.NullQ(1) || a.NullQ(0) {
		t.Error("NullQ")
	}
	if a.NullQ(99) != true {
		t.Error("NullQ out-of-range should be true")
	}
	if a.NullQ(-1) {
		t.Error("NullQ(-1) points at valid last elem")
	}
	if !a.ValidQ(0) || a.ValidQ(1) {
		t.Error("ValidQ")
	}
	if a.ValidQ(99) {
		t.Error("ValidQ out-of-range should be false")
	}
	if !a.ValidQ(-1) {
		t.Error("ValidQ(-1) should be valid")
	}
}

func TestArrayEach(t *testing.T) {
	a := mustArray(t, Int64(), []any{int64(1), int64(2), int64(3)})
	sum := int64(0)
	if err := a.Each(func(_ int, v any) error {
		sum += v.(int64)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if sum != 6 {
		t.Fatalf("sum %d", sum)
	}
	boom := errors.New("boom")
	count := 0
	if err := a.Each(func(_ int, _ any) error {
		count++
		return boom
	}); err != boom {
		t.Fatalf("Each error propagation: %v", err)
	}
	if count != 1 {
		t.Fatalf("Each should stop at first error, ran %d", count)
	}
}

func TestArrayBuilderChaining(t *testing.T) {
	b := NewArrayBuilder(Int64())
	if b.DataType().Name() != "int64" {
		t.Error("builder datatype")
	}
	if err := b.Append(int64(1)); err != nil {
		t.Fatal(err)
	}
	b.AppendNull()
	if b.Length() != 2 {
		t.Fatalf("builder length %d", b.Length())
	}
	a := b.Finish()
	if a.Length() != 2 {
		t.Fatalf("finished length %d", a.Length())
	}
}

func TestNewArrayInference(t *testing.T) {
	cases := []struct {
		in   []any
		name string
	}{
		{[]any{nil, true}, "bool"},
		{[]any{"x"}, "utf8"},
		{[]any{1.5}, "float64"},
		{[]any{int(3)}, "int64"},
		{[]any{uint(3)}, "uint64"},
		{[]any{time.Now()}, "timestamp"},
	}
	for _, tc := range cases {
		a, err := NewArray(tc.in)
		if err != nil {
			t.Fatalf("NewArray(%v): %v", tc.in, err)
		}
		if a.DataType().Name() != tc.name {
			t.Errorf("infer %v -> %s want %s", tc.in, a.DataType().Name(), tc.name)
		}
	}
}

func TestNewArrayInferenceErrors(t *testing.T) {
	if _, err := NewArray([]any{struct{}{}}); !errors.Is(err, ErrType) {
		t.Errorf("unsupported type err=%v", err)
	}
	if _, err := NewArray([]any{}); !errors.Is(err, ErrArgument) {
		t.Errorf("empty err=%v", err)
	}
	if _, err := NewArray([]any{nil, nil}); !errors.Is(err, ErrArgument) {
		t.Errorf("all-nil err=%v", err)
	}
}

func TestAppendTypeErrors(t *testing.T) {
	typeErr := func(dt *DataType, v any) {
		t.Helper()
		if _, err := NewArrayOf(dt, []any{v}); !errors.Is(err, ErrType) {
			t.Errorf("%s <- %T: err=%v", dt.Name(), v, err)
		}
	}
	typeErr(Boolean(), "no")
	typeErr(Int8(), "no")
	typeErr(Int16(), "no")
	typeErr(Int32(), "no")
	typeErr(Int64(), "no")
	typeErr(UInt8(), "no")
	typeErr(UInt16(), "no")
	typeErr(UInt32(), "no")
	typeErr(UInt64(), "no")
	typeErr(Float64(), "no")
	typeErr(Float32(), "no")
	typeErr(StringType(), 1)
	typeErr(Timestamp(), "no")
	typeErr(Date(), "no")
	typeErr(Decimal128(5, 2), true)
	typeErr(ListOf(Int64()), "no")
	typeErr(StructOf(NewField("a", Int64())), "no")
	// negative into unsigned -> type error (asUint64 rejects)
	typeErr(UInt8(), -1)
}

func TestAppendRangeErrors(t *testing.T) {
	rangeErr := func(dt *DataType, v any) {
		t.Helper()
		if _, err := NewArrayOf(dt, []any{v}); !errors.Is(err, ErrArgument) {
			t.Errorf("%s <- %v: err=%v", dt.Name(), v, err)
		}
	}
	rangeErr(Int8(), 1000)
	rangeErr(UInt8(), 1000)
	// decimal parse error
	rangeErr(Decimal128(5, 2), "not-a-number")
}

func TestAppendNestedError(t *testing.T) {
	// a bad element inside a list surfaces the element's type error
	if _, err := NewArrayOf(ListOf(Int64()), []any{[]any{"bad"}}); !errors.Is(err, ErrType) {
		t.Errorf("nested list err=%v", err)
	}
	// a bad field value inside a struct surfaces too
	if _, err := NewArrayOf(StructOf(NewField("a", Int64())),
		[]any{map[string]any{"a": "bad"}}); !errors.Is(err, ErrType) {
		t.Errorf("nested struct err=%v", err)
	}
}

func TestNewArrayOfPropagatesError(t *testing.T) {
	if _, err := NewArrayOf(Int64(), []any{int64(1), "bad"}); err == nil {
		t.Fatal("expected error from bad value")
	}
}

func TestCoercionHelpers(t *testing.T) {
	// asInt64 across integer kinds via an Int64 array
	ints := []any{int(1), int8(2), int16(3), int32(4), int64(5), uint8(6), uint16(7), uint32(8), uint(9), uint64(10)}
	a := mustArray(t, Int64(), ints)
	if a.Length() != len(ints) {
		t.Fatalf("len %d", a.Length())
	}
	// uint overflow into signed -> type error
	if _, err := NewArrayOf(Int64(), []any{uint64(math.MaxUint64)}); !errors.Is(err, ErrType) {
		t.Errorf("uint64 overflow err=%v", err)
	}
	if _, err := NewArrayOf(Int64(), []any{uint(math.MaxUint64)}); !errors.Is(err, ErrType) {
		t.Errorf("uint overflow err=%v", err)
	}
	// asUint64 large value ok
	if _, err := NewArrayOf(UInt64(), []any{uint64(math.MaxUint64)}); err != nil {
		t.Errorf("uint64 max into UInt64: %v", err)
	}
	// asFloat64 from int
	fa := mustArray(t, Float64(), []any{int(3)})
	fv, _ := fa.Get(0)
	if fv.(float64) != 3 {
		t.Errorf("float from int %v", fv)
	}
}

func TestUnsupportedBuilderAndReader(t *testing.T) {
	// appendValue default branch: a Time32 builder is not mapped.
	time32 := FromArrowType(&xarrow.Time32Type{Unit: xarrow.Second})
	b := NewArrayBuilder(time32)
	if err := b.Append(int32(1)); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("unsupported append err=%v", err)
	}
	b.Release()

	// getValue default branch: read a Time32 array (built with arrow-go directly).
	tb := array.NewTime32Builder(alloc, &xarrow.Time32Type{Unit: xarrow.Second})
	if err := tb.AppendValueFromString("00:00:01"); err != nil {
		t.Fatal(err)
	}
	arr := &Array{arr: tb.NewArray()}
	v, _ := arr.Get(0)
	if _, ok := v.(string); !ok {
		t.Errorf("Time32 fallback should be a string, got %T", v)
	}
}
