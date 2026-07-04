// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

package arrow

import xarrow "github.com/apache/arrow-go/v18/arrow"

// DataType is the pure-Go counterpart of Ruby's Arrow::DataType. It wraps an
// arrow-go [xarrow.DataType] and re-presents it with Ruby naming. Construct one
// with the type constructors ([Int8], [StringType], [ListOf], [StructOf], …).
type DataType struct {
	dt xarrow.DataType
}

func wrapDataType(dt xarrow.DataType) *DataType { return &DataType{dt: dt} }

// Unwrap returns the underlying arrow-go data type.
func (d *DataType) Unwrap() xarrow.DataType { return d.dt }

// Name returns the Arrow type name (e.g. "int64", "utf8"), mirroring
// Arrow::DataType#name.
func (d *DataType) Name() string { return d.dt.Name() }

// String returns the type's canonical string form (Arrow::DataType#to_s).
func (d *DataType) String() string { return d.dt.String() }

// EqualQ reports whether two data types are equal (Arrow::DataType#==).
func (d *DataType) EqualQ(other *DataType) bool {
	return xarrow.TypeEqual(d.dt, other.dt)
}

// Primitive type constructors mirror red-arrow's Arrow::<T>DataType classes.

// Int8 returns the 8-bit signed integer type (Arrow::Int8DataType).
func Int8() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Int8) }

// Int16 returns the 16-bit signed integer type.
func Int16() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Int16) }

// Int32 returns the 32-bit signed integer type.
func Int32() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Int32) }

// Int64 returns the 64-bit signed integer type.
func Int64() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Int64) }

// UInt8 returns the 8-bit unsigned integer type (Arrow::UInt8DataType).
func UInt8() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Uint8) }

// UInt16 returns the 16-bit unsigned integer type.
func UInt16() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Uint16) }

// UInt32 returns the 32-bit unsigned integer type.
func UInt32() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Uint32) }

// UInt64 returns the 64-bit unsigned integer type.
func UInt64() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Uint64) }

// Float32 returns the single-precision float type (Arrow::FloatDataType).
func Float32() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Float32) }

// Float64 returns the double-precision float type (Arrow::DoubleDataType).
func Float64() *DataType { return wrapDataType(xarrow.PrimitiveTypes.Float64) }

// Float is an alias of [Float64], matching red-arrow's default for Ruby Floats.
func Float() *DataType { return Float64() }

// Boolean returns the boolean type (Arrow::BooleanDataType).
func Boolean() *DataType { return wrapDataType(xarrow.FixedWidthTypes.Boolean) }

// StringType returns the UTF-8 string type (Arrow::StringDataType).
func StringType() *DataType { return wrapDataType(xarrow.BinaryTypes.String) }

// Timestamp returns a microsecond, timezone-naive timestamp type
// (Arrow::TimestampDataType), the default red-arrow infers for Ruby Time.
func Timestamp() *DataType {
	return wrapDataType(&xarrow.TimestampType{Unit: xarrow.Microsecond})
}

// Date returns the 32-bit date type — days since the Unix epoch
// (Arrow::Date32DataType).
func Date() *DataType { return wrapDataType(xarrow.FixedWidthTypes.Date32) }

// Decimal128 returns a 128-bit fixed-point decimal type with the given
// precision and scale (Arrow::Decimal128DataType).
func Decimal128(precision, scale int32) *DataType {
	return wrapDataType(&xarrow.Decimal128Type{Precision: precision, Scale: scale})
}

// ListOf returns a list type whose elements have type elem
// (Arrow::ListDataType).
func ListOf(elem *DataType) *DataType {
	return wrapDataType(xarrow.ListOf(elem.dt))
}

// StructOf returns a struct type composed of the given fields
// (Arrow::StructDataType).
func StructOf(fields ...*Field) *DataType {
	xf := make([]xarrow.Field, len(fields))
	for i, f := range fields {
		xf[i] = f.f
	}
	return wrapDataType(xarrow.StructOf(xf...))
}

// Field is the pure-Go counterpart of Arrow::Field — a named, nullable slot in
// a [Schema] or struct type.
type Field struct {
	f xarrow.Field
}

// NewField returns a nullable field named name of type dt (Arrow::Field.new).
func NewField(name string, dt *DataType) *Field {
	return &Field{f: xarrow.Field{Name: name, Type: dt.dt, Nullable: true}}
}

// NewFieldNonNull returns a non-nullable field named name of type dt.
func NewFieldNonNull(name string, dt *DataType) *Field {
	return &Field{f: xarrow.Field{Name: name, Type: dt.dt, Nullable: false}}
}

func wrapField(f xarrow.Field) *Field { return &Field{f: f} }

// Unwrap returns the underlying arrow-go field.
func (f *Field) Unwrap() xarrow.Field { return f.f }

// Name returns the field name (Arrow::Field#name).
func (f *Field) Name() string { return f.f.Name }

// DataType returns the field's data type (Arrow::Field#data_type).
func (f *Field) DataType() *DataType { return wrapDataType(f.f.Type) }

// NullableQ reports whether the field admits nulls (Arrow::Field#nullable?).
func (f *Field) NullableQ() bool { return f.f.Nullable }

// String returns the field's canonical string form.
func (f *Field) String() string { return f.f.String() }

// Schema is the pure-Go counterpart of Arrow::Schema — an ordered list of
// [Field]s describing a [RecordBatch] or [Table].
type Schema struct {
	s *xarrow.Schema
}

// NewSchema builds a schema from the given fields (Arrow::Schema.new).
func NewSchema(fields ...*Field) *Schema {
	xf := make([]xarrow.Field, len(fields))
	for i, f := range fields {
		xf[i] = f.f
	}
	return &Schema{s: xarrow.NewSchema(xf, nil)}
}

func wrapSchema(s *xarrow.Schema) *Schema { return &Schema{s: s} }

// Unwrap returns the underlying arrow-go schema.
func (s *Schema) Unwrap() *xarrow.Schema { return s.s }

// NumFields returns the number of fields (Arrow::Schema#n_fields).
func (s *Schema) NumFields() int { return s.s.NumFields() }

// Field returns the i-th field (Arrow::Schema#[] by index).
func (s *Schema) Field(i int) *Field { return wrapField(s.s.Field(i)) }

// FieldByName returns the field named name and whether it was found
// (Arrow::Schema#[] by name).
func (s *Schema) FieldByName(name string) (*Field, bool) {
	fs := s.s.FieldIndices(name)
	if len(fs) == 0 {
		return nil, false
	}
	return wrapField(s.s.Field(fs[0])), true
}

// Fields returns all fields in order (Arrow::Schema#fields).
func (s *Schema) Fields() []*Field {
	xf := s.s.Fields()
	out := make([]*Field, len(xf))
	for i := range xf {
		out[i] = wrapField(xf[i])
	}
	return out
}

// String returns the schema's canonical string form (Arrow::Schema#to_s).
func (s *Schema) String() string { return s.s.String() }
