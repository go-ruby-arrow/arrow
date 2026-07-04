// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright (c) 2026, the go-ruby-arrow/arrow authors

// Package arrow is a pure-Go (CGO=0), MRI-faithful implementation of the Ruby
// red-arrow gem's core surface — Apache Arrow's columnar in-memory format and
// its IPC (Feather / Arrow stream & file) serialization.
//
// # Relationship to upstream
//
// The real red-arrow gem is a thin Ruby binding over the C library libarrow
// (via GObject introspection), so it cannot be shipped in a CGO-free static
// binary. This package mirrors red-arrow's observable Ruby surface — Array,
// ArrayBuilder, DataType, Field, Schema, RecordBatch, Table and the IPC
// round-trip — on top of [github.com/apache/arrow-go/v18], the official
// pure-Go Apache Arrow implementation. It does not reimplement the columnar
// format itself; it re-presents arrow-go through Ruby's naming and semantics
// so it can back an embedded Ruby (go-embedded-ruby / rbgo) with no cgo.
//
// # Ruby-to-Go mapping
//
//	Arrow::Array          -> *Array          (Enumerable via Each/ToSlice)
//	Arrow::ArrayBuilder   -> *ArrayBuilder
//	Arrow::DataType       -> *DataType       (Int8()..Decimal128()/ListOf/StructOf)
//	Arrow::Field          -> *Field
//	Arrow::Schema         -> *Schema
//	Arrow::RecordBatch    -> *RecordBatch
//	Arrow::Table          -> *Table
//	Arrow::Error (tree)   -> *Error          (Kind + RubyClass mapping)
//
// Ruby predicate methods ending in "?" map to Go methods ending in "Q"
// (IsNull -> null?, etc.), and mutating "!" methods are spelled out. Ruby
// integer/float towers collapse onto Go's int64/float64 at the boundary, with
// explicit typed builders preserving Arrow's Int8..Int64 / UInt8..UInt64 /
// Float32/Float64 / Boolean / String / Timestamp / Date32 / Decimal128 / List
// / Struct widths.
//
// # IPC wire compatibility
//
// Tables and record batches round-trip through both the Arrow IPC streaming
// format ([WriteTableStream] / [ReadTableStream]) and the Arrow IPC file
// (a.k.a. Feather v2) format ([WriteTableFile] / [ReadTableFile]). The bytes
// this package emits are the bytes arrow-go's canonical ipc.Reader/FileReader
// consume, and vice-versa — verified by the differential tests, not asserted.
// Arrow buffers are little-endian on the wire on every architecture; on
// big-endian targets (s390x) arrow-go performs the byte swap, so the same
// bytes round-trip identically across all six supported 64-bit arches.
//
// # Scope
//
// This covers red-arrow's Array/Schema/Table/RecordBatch core plus IPC. It does
// not (yet) cover red-arrow's compute kernels, Parquet/CSV readers, Datasets,
// Flight, or the full CategoricalArray/DictionaryArray DSL; those are additive
// follow-ups on the same arrow-go foundation.
package arrow
