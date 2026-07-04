// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import (
	"math"
	"time"

	xarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// alloc is the shared, CGO-free Go allocator. arrow-go's DefaultAllocator uses a
// cgo mallocator; NewGoAllocator keeps this package pure-Go (CGO=0).
var alloc memory.Allocator = memory.NewGoAllocator()

// FromArrowType wraps an arbitrary arrow-go data type as a [DataType]. It is the
// escape hatch for types this package's constructors do not spell out (for
// example a type recovered from an externally-produced IPC stream).
func FromArrowType(dt xarrow.DataType) *DataType { return wrapDataType(dt) }

// Array is the pure-Go counterpart of Arrow::Array — an immutable, typed,
// nullable column. Build one with [NewArray] (type inferred), [NewArrayOf]
// (explicit type), or an [ArrayBuilder]. It is Enumerable via [Array.Each] and
// [Array.ToSlice].
type Array struct {
	arr xarrow.Array
}

func wrapArray(a xarrow.Array) *Array { return &Array{arr: a} }

// Unwrap returns the underlying arrow-go array.
func (a *Array) Unwrap() xarrow.Array { return a.arr }

// DataType returns the array's element type (Arrow::Array#value_data_type).
func (a *Array) DataType() *DataType { return wrapDataType(a.arr.DataType()) }

// Length returns the number of elements (Arrow::Array#length / #n_rows).
func (a *Array) Length() int { return a.arr.Len() }

// NNulls returns the number of null elements (Arrow::Array#n_nulls).
func (a *Array) NNulls() int { return a.arr.NullN() }

// NullQ reports whether the element at index i is null (Arrow::Array#null?).
// It supports Ruby-style negative indexing.
func (a *Array) NullQ(i int) bool {
	n := a.arr.Len()
	if i < 0 {
		i += n
	}
	if i < 0 || i >= n {
		return true
	}
	return a.arr.IsNull(i)
}

// ValidQ reports whether the element at index i is non-null
// (Arrow::Array#valid?).
func (a *Array) ValidQ(i int) bool {
	n := a.arr.Len()
	if i < 0 {
		i += n
	}
	if i < 0 || i >= n {
		return false
	}
	return a.arr.IsValid(i)
}

// Get returns the element at index i (Arrow::Array#[]). It supports Ruby-style
// negative indexing and returns an [*Error] of [KindIndex] when out of range. A
// null element reads back as nil.
func (a *Array) Get(i int) (any, error) {
	n := a.arr.Len()
	if i < 0 {
		i += n
	}
	if i < 0 || i >= n {
		return nil, newError(KindIndex, "index %d out of range (length %d)", i, n)
	}
	return getValue(a.arr, i), nil
}

// ToSlice returns all elements as a Go slice (Arrow::Array#to_a), nulls as nil.
func (a *Array) ToSlice() []any {
	n := a.arr.Len()
	out := make([]any, n)
	for i := 0; i < n; i++ {
		out[i] = getValue(a.arr, i)
	}
	return out
}

// Each iterates the elements in order (Arrow::Array#each, Enumerable). It stops
// and returns the first error fn yields.
func (a *Array) Each(fn func(i int, v any) error) error {
	n := a.arr.Len()
	for i := 0; i < n; i++ {
		if err := fn(i, getValue(a.arr, i)); err != nil {
			return err
		}
	}
	return nil
}

// String returns the array's canonical string form (Arrow::Array#to_s).
func (a *Array) String() string { return a.arr.String() }

// Release drops the array's reference to its buffers.
func (a *Array) Release() { a.arr.Release() }

// ArrayBuilder is the pure-Go counterpart of Arrow::ArrayBuilder — an appender
// that accumulates typed values (and nulls) and freezes them into an [Array].
type ArrayBuilder struct {
	dt      *DataType
	builder array.Builder
}

// NewArrayBuilder returns a builder for the given element type
// (Arrow::ArrayBuilder.build's typed builder).
func NewArrayBuilder(dt *DataType) *ArrayBuilder {
	return &ArrayBuilder{dt: dt, builder: array.NewBuilder(alloc, dt.dt)}
}

// DataType returns the element type this builder produces.
func (b *ArrayBuilder) DataType() *DataType { return b.dt }

// Length returns the number of values appended so far.
func (b *ArrayBuilder) Length() int { return b.builder.Len() }

// Append appends one value, coercing Go scalars to the column type. A nil value
// appends a null (Arrow::ArrayBuilder#append_value / #append_null).
func (b *ArrayBuilder) Append(v any) error { return appendValue(b.builder, v) }

// AppendNull appends a null and returns the builder for chaining.
func (b *ArrayBuilder) AppendNull() *ArrayBuilder {
	b.builder.AppendNull()
	return b
}

// AppendValues appends each value in order, stopping at the first error
// (Arrow::ArrayBuilder#append_values).
func (b *ArrayBuilder) AppendValues(values []any) error {
	for _, v := range values {
		if err := appendValue(b.builder, v); err != nil {
			return err
		}
	}
	return nil
}

// Finish freezes the accumulated values into an [Array] and resets the builder
// (Arrow::ArrayBuilder#finish).
func (b *ArrayBuilder) Finish() *Array { return wrapArray(b.builder.NewArray()) }

// Release frees the builder's working buffers.
func (b *ArrayBuilder) Release() { b.builder.Release() }

// NewArrayOf builds an [Array] of the explicit type dt from values
// (Arrow::Array.new(values, type: dt)).
func NewArrayOf(dt *DataType, values []any) (*Array, error) {
	b := NewArrayBuilder(dt)
	if err := b.AppendValues(values); err != nil {
		b.Release()
		return nil, err
	}
	return b.Finish(), nil
}

// NewArray builds an [Array] from values, inferring the element type from the
// first non-null value (Arrow::Array.new(values)). Bool infers Boolean, string
// infers String, floats infer Float64, signed ints infer Int64, unsigned ints
// infer UInt64, and time.Time infers Timestamp. An empty or all-null slice is an
// error, as the type cannot be inferred.
func NewArray(values []any) (*Array, error) {
	dt, err := inferType(values)
	if err != nil {
		return nil, err
	}
	return NewArrayOf(dt, values)
}

func inferType(values []any) (*DataType, error) {
	for _, v := range values {
		switch v.(type) {
		case nil:
			continue
		case bool:
			return Boolean(), nil
		case string:
			return StringType(), nil
		case float32, float64:
			return Float64(), nil
		case int, int8, int16, int32, int64:
			return Int64(), nil
		case uint, uint8, uint16, uint32, uint64:
			return UInt64(), nil
		case time.Time:
			return Timestamp(), nil
		default:
			return nil, newError(KindType, "cannot infer Arrow type from %T", v)
		}
	}
	return nil, newError(KindArgument, "cannot infer Arrow type from empty or all-null values")
}

// appendValue coerces a Go scalar onto an arrow-go builder of any supported
// type. A nil value appends a null on every builder.
func appendValue(b array.Builder, v any) error {
	if v == nil {
		b.AppendNull()
		return nil
	}
	switch tb := b.(type) {
	case *array.BooleanBuilder:
		x, ok := v.(bool)
		if !ok {
			return newError(KindType, "cannot convert %T to Boolean", v)
		}
		tb.Append(x)
	case *array.Int8Builder:
		n, err := signedInRange(v, math.MinInt8, math.MaxInt8, "Int8")
		if err != nil {
			return err
		}
		tb.Append(int8(n))
	case *array.Int16Builder:
		n, err := signedInRange(v, math.MinInt16, math.MaxInt16, "Int16")
		if err != nil {
			return err
		}
		tb.Append(int16(n))
	case *array.Int32Builder:
		n, err := signedInRange(v, math.MinInt32, math.MaxInt32, "Int32")
		if err != nil {
			return err
		}
		tb.Append(int32(n))
	case *array.Int64Builder:
		n, err := signedInRange(v, math.MinInt64, math.MaxInt64, "Int64")
		if err != nil {
			return err
		}
		tb.Append(n)
	case *array.Uint8Builder:
		n, err := unsignedInRange(v, math.MaxUint8, "UInt8")
		if err != nil {
			return err
		}
		tb.Append(uint8(n))
	case *array.Uint16Builder:
		n, err := unsignedInRange(v, math.MaxUint16, "UInt16")
		if err != nil {
			return err
		}
		tb.Append(uint16(n))
	case *array.Uint32Builder:
		n, err := unsignedInRange(v, math.MaxUint32, "UInt32")
		if err != nil {
			return err
		}
		tb.Append(uint32(n))
	case *array.Uint64Builder:
		n, err := unsignedInRange(v, math.MaxUint64, "UInt64")
		if err != nil {
			return err
		}
		tb.Append(n)
	case *array.Float64Builder:
		f, ok := asFloat64(v)
		if !ok {
			return newError(KindType, "cannot convert %T to Float64", v)
		}
		tb.Append(f)
	case *array.Float32Builder:
		f, ok := asFloat64(v)
		if !ok {
			return newError(KindType, "cannot convert %T to Float32", v)
		}
		tb.Append(float32(f))
	case *array.StringBuilder:
		x, ok := v.(string)
		if !ok {
			return newError(KindType, "cannot convert %T to String", v)
		}
		tb.Append(x)
	case *array.TimestampBuilder:
		return appendTimestamp(tb, v)
	case *array.Date32Builder:
		return appendDate32(tb, v)
	case *array.Decimal128Builder:
		return appendDecimal(tb, v)
	case *array.ListBuilder:
		return appendList(tb, v)
	case *array.StructBuilder:
		return appendStruct(tb, v)
	default:
		return newError(KindNotImplemented, "appending to %s is not implemented", b.Type().Name())
	}
	return nil
}

func appendTimestamp(tb *array.TimestampBuilder, v any) error {
	unit := tb.Type().(*xarrow.TimestampType).Unit
	switch x := v.(type) {
	case time.Time:
		// All four TimeUnit values are valid, and this package only builds
		// Microsecond timestamps, so the conversion never fails here.
		ts, _ := xarrow.TimestampFromTime(x, unit)
		tb.Append(ts)
	case int64:
		tb.Append(xarrow.Timestamp(x))
	case int:
		tb.Append(xarrow.Timestamp(int64(x)))
	default:
		return newError(KindType, "cannot convert %T to Timestamp", v)
	}
	return nil
}

func appendDate32(tb *array.Date32Builder, v any) error {
	switch x := v.(type) {
	case time.Time:
		tb.Append(xarrow.Date32FromTime(x))
	case int32:
		tb.Append(xarrow.Date32(x))
	case int:
		tb.Append(xarrow.Date32(int32(x)))
	default:
		return newError(KindType, "cannot convert %T to Date32", v)
	}
	return nil
}

func appendDecimal(tb *array.Decimal128Builder, v any) error {
	dt := tb.Type().(*xarrow.Decimal128Type)
	d, ok, err := toDecimal(v, dt.Precision, dt.Scale)
	if !ok {
		return newError(KindType, "cannot convert %T to Decimal128", v)
	}
	if err != nil {
		return wrapError(KindArgument, err, "invalid Decimal128 value %v", v)
	}
	tb.Append(d)
	return nil
}

func toDecimal(v any, prec, scale int32) (decimal128.Num, bool, error) {
	switch x := v.(type) {
	case string:
		d, err := decimal128.FromString(x, prec, scale)
		return d, true, err
	case float64:
		d, err := decimal128.FromFloat64(x, prec, scale)
		return d, true, err
	case float32:
		d, err := decimal128.FromFloat64(float64(x), prec, scale)
		return d, true, err
	default:
		return decimal128.Num{}, false, nil
	}
}

func appendList(tb *array.ListBuilder, v any) error {
	inner, ok := v.([]any)
	if !ok {
		return newError(KindType, "cannot convert %T to List", v)
	}
	tb.Append(true)
	vb := tb.ValueBuilder()
	for _, e := range inner {
		if err := appendValue(vb, e); err != nil {
			return err
		}
	}
	return nil
}

func appendStruct(tb *array.StructBuilder, v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return newError(KindType, "cannot convert %T to Struct", v)
	}
	tb.Append(true)
	st := tb.Type().(*xarrow.StructType)
	for i := 0; i < tb.NumField(); i++ {
		if err := appendValue(tb.FieldBuilder(i), m[st.Field(i).Name]); err != nil {
			return err
		}
	}
	return nil
}

// signedInRange coerces v to an int64 and checks it fits [min, max].
func signedInRange(v any, min, max int64, name string) (int64, error) {
	n, ok := asInt64(v)
	if !ok {
		return 0, newError(KindType, "cannot convert %T to %s", v, name)
	}
	if n < min || n > max {
		return 0, newError(KindArgument, "%d out of range for %s", n, name)
	}
	return n, nil
}

// unsignedInRange coerces v to a uint64 and checks it does not exceed max.
func unsignedInRange(v any, max uint64, name string) (uint64, error) {
	n, ok := asUint64(v)
	if !ok {
		return 0, newError(KindType, "cannot convert %T to %s", v, name)
	}
	if n > max {
		return 0, newError(KindArgument, "%d out of range for %s", n, name)
	}
	return n, nil
}

func asInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint:
		if uint64(x) > math.MaxInt64 {
			return 0, false
		}
		return int64(x), true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		if x > math.MaxInt64 {
			return 0, false
		}
		return int64(x), true
	default:
		return 0, false
	}
}

func asUint64(v any) (uint64, bool) {
	switch x := v.(type) {
	case uint64:
		return x, true
	case uint:
		return uint64(x), true
	default:
		n, ok := asInt64(v)
		if !ok || n < 0 {
			return 0, false
		}
		return uint64(n), true
	}
}

func asFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	default:
		if n, ok := asInt64(v); ok {
			return float64(n), true
		}
		return 0, false
	}
}

// getValue reads the native Go value at index i of an arrow-go array, mapping
// each Arrow type to the faithful Go/Ruby scalar. Nulls read back as nil.
func getValue(arr xarrow.Array, i int) any {
	if arr.IsNull(i) {
		return nil
	}
	switch a := arr.(type) {
	case *array.Boolean:
		return a.Value(i)
	case *array.Int8:
		return a.Value(i)
	case *array.Int16:
		return a.Value(i)
	case *array.Int32:
		return a.Value(i)
	case *array.Int64:
		return a.Value(i)
	case *array.Uint8:
		return a.Value(i)
	case *array.Uint16:
		return a.Value(i)
	case *array.Uint32:
		return a.Value(i)
	case *array.Uint64:
		return a.Value(i)
	case *array.Float32:
		return a.Value(i)
	case *array.Float64:
		return a.Value(i)
	case *array.String:
		return a.Value(i)
	case *array.Timestamp:
		unit := a.DataType().(*xarrow.TimestampType).Unit
		return a.Value(i).ToTime(unit).UTC()
	case *array.Date32:
		return a.Value(i).ToTime().UTC()
	case *array.Decimal128:
		scale := a.DataType().(*xarrow.Decimal128Type).Scale
		return a.Value(i).ToString(scale)
	case *array.List:
		return listValues(a, i)
	case *array.Struct:
		return structValues(a, i)
	default:
		return arr.ValueStr(i)
	}
}

func listValues(a *array.List, i int) []any {
	start, end := a.ValueOffsets(i)
	vals := a.ListValues()
	out := make([]any, 0, end-start)
	for j := start; j < end; j++ {
		out = append(out, getValue(vals, int(j)))
	}
	return out
}

func structValues(a *array.Struct, i int) map[string]any {
	st := a.DataType().(*xarrow.StructType)
	out := make(map[string]any, a.NumField())
	for f := 0; f < a.NumField(); f++ {
		out[st.Field(f).Name] = getValue(a.Field(f), i)
	}
	return out
}
