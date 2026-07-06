<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-arrow/brand/main/social/go-ruby-arrow-arrow.png" alt="go-ruby-arrow/arrow" width="720"></p>

# arrow — go-ruby-arrow

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-arrow.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's [`red-arrow`](https://arrow.apache.org/docs/ruby/) gem**
— the Apache Arrow columnar in-memory format and its IPC serialization. The real
`red-arrow` gem binds the C library **libarrow** through GObject introspection,
so it cannot ship inside a CGO-free static binary. This package mirrors
`red-arrow`'s observable Ruby surface — `Arrow::Array`, `Arrow::ArrayBuilder`,
`Arrow::DataType`/`Field`/`Schema`, `Arrow::RecordBatch`, `Arrow::Table`, and the
IPC round-trip — on top of [`github.com/apache/arrow-go/v18`](https://github.com/apache/arrow-go),
the official pure-Go Apache Arrow implementation. It **does not reimplement the
columnar format**; it re-presents arrow-go through Ruby's naming and semantics.

It is the `Arrow` backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module — a sibling of
[go-ruby-marshal](https://github.com/go-ruby-marshal/marshal) and
[go-ruby-msgpack](https://github.com/go-ruby-msgpack/msgpack).

> **Consumes, does not reinvent.** Arrays, schema, record batches, IPC readers and
> writers all come from `arrow-go`. The value this package adds is the faithful
> Ruby surface and the Go↔Ruby scalar mapping, verified wire-compatible with
> `arrow-go`'s canonical `ipc.Reader`/`FileReader` in both directions.

## Install

```sh
go get github.com/go-ruby-arrow/arrow
```

## Usage

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/go-ruby-arrow/arrow"
)

func main() {
	schema := arrow.NewSchema(
		arrow.NewField("id", arrow.Int64()),
		arrow.NewField("name", arrow.StringType()),
	)
	id, _ := arrow.NewArrayOf(arrow.Int64(), []any{int64(1), int64(2), nil})
	name, _ := arrow.NewArrayOf(arrow.StringType(), []any{"a", "b", "c"})

	table, _ := arrow.NewTable(schema, []*arrow.Array{id, name})
	fmt.Println(table.NumRows(), table.NumColumns()) // 3 2

	// Arrow IPC round-trip (bytes stable, values preserved).
	var buf bytes.Buffer
	_ = arrow.WriteTableStream(&buf, table)
	back, _ := arrow.ReadTableStream(bytes.NewReader(buf.Bytes()))

	col, _ := back.Column("name")
	v, _ := col.Get(2)
	fmt.Println(v) // c
}
```

## Ruby-to-Go mapping

| Ruby (`red-arrow`)      | Go (this package) |
| ----------------------- | ----------------- |
| `Arrow::Array`          | `*Array` — `Get` (`#[]`), `Length`, `NullQ`, `ToSlice` (`#to_a`), `Each` |
| `Arrow::ArrayBuilder`   | `*ArrayBuilder` — `Append`, `AppendNull`, `Finish` |
| `Arrow::DataType`       | `*DataType` — `Int8()`…`Int64()`/`UInt8()`…/`Float64()`/`Boolean()`/`StringType()`/`Timestamp()`/`Date()`/`Decimal128()`/`ListOf()`/`StructOf()` |
| `Arrow::Field`          | `*Field` |
| `Arrow::Schema`         | `*Schema` |
| `Arrow::RecordBatch`    | `*RecordBatch` — `NumRows`, `NumColumns`, `Get` (`#[]`), `Slice`, `ToHash`, `EachRecord` |
| `Arrow::Table`          | `*Table` — same plus `NewTableFromRecordBatches`, `ConcatTables`, `Save`/`LoadTable` |
| `Arrow::Error` tree     | `*Error` (`Kind` + `RubyClass()`) |

Ruby predicate methods (`null?`, `valid?`) map to Go `…Q` methods; Ruby's
integer/float towers collapse onto Go `int64`/`float64` at the boundary, with the
typed builders preserving Arrow's `Int8`..`Int64` / `UInt8`..`UInt64` /
`Float32`/`Float64` / `Boolean` / `String` / `Timestamp` / `Date32` /
`Decimal128` / `List` / `Struct` widths.

## IPC wire compatibility

Tables round-trip through both the Arrow IPC **streaming** format
(`WriteTableStream` / `ReadTableStream`) and the Arrow IPC **file** / Feather v2
format (`WriteTableFile` / `ReadTableFile`). The bytes this package emits are the
bytes `arrow-go`'s canonical `ipc.Reader`/`FileReader` consume, and vice-versa —
this is **verified** by the tests (encode here → decode with `arrow-go`, and
encode with `arrow-go` → decode here), not asserted. Arrow buffers are
little-endian on the wire on every architecture; on big-endian targets (s390x)
`arrow-go` performs the byte swap, so identical bytes round-trip across all six
supported 64-bit arches.

## Scope

This covers `red-arrow`'s Array/Schema/Table/RecordBatch core plus IPC. It does
not (yet) cover compute kernels, Parquet/CSV readers, Datasets, Flight, or the
full Dictionary/Categorical DSL; those are additive follow-ups on the same
`arrow-go` foundation. See [`doc.go`](doc.go) for the authoritative scope note.

## Tests & coverage

The suite is deterministic and dependency-light (no libarrow, **CGO=0**): typed
build/read for every listed type including nulls, negative indexing, the full
error tree, and IPC round-trips in both formats — plus the cross-checks against
`arrow-go`'s canonical readers/writers that pin wire compatibility.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

CGO-free, `gofmt` + `go vet` clean, and green across the six 64-bit Go targets
(amd64, arm64, riscv64, loong64, ppc64le, s390x — the last big-endian) and three
OSes (Linux, macOS, Windows).

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-arrow/arrow authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
