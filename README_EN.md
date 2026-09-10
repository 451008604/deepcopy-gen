# deepcopy-gen

[中文](README.md)

Generate deep-copy methods for Go structs and locally defined named containers, without writing repetitive copy code.

## Features

- Supports pointers, slices, maps, arrays, nested combinations, type aliases, and generic types.
- Preserves nil values, non-nil empty containers, repeated pointer references, and cycles formed through pointers.
- Supports cross-package types, preferring `DeepCopyWithVisited` to share visited state and also recognizing `DeepCopy()`.
- Generates code without runtime reflection, with optimized copy paths for basic containers and value-only objects.
- Scans subdirectories recursively, writes `deepcopy.dc.go` per package, and avoids declaration name collisions.

## Advantages

- Reduces handwritten copy logic and the risk of missing fields; regenerate after type changes.
- Produces ordinary Go code that can be reviewed, debugged, and checked by the compiler alongside your project.
- Avoids reflection and serialization round trips, using direct copy paths for basic containers and value-only objects.

## Installation

Requires Go 1.21 or later. Ensure the Go tool installation directory is on your `PATH`.

```sh
go install github.com/451008604/deepcopy-gen@latest
```

## Usage

Run in your target project:

```sh
# Generate copy methods for a directory and its subdirectories
deepcopy-gen ./models

# Preview generated code without writing files
deepcopy-gen -dry-run ./models

# Print the tool version
deepcopy-gen -version
```

Specify the directory as a positional argument, with options before it. The `-dir` and `-v` flags have been removed. Use `-h` for help.

Version output uses the module version from Go build metadata when available. Otherwise, it reports `dev`, including the Git revision and modification status when available.

Call `DeepCopy()` on your object after generation:

```go
cloned := original.DeepCopy()
```

To preserve shared pointers across multiple root objects, pass the same visited map to their `DeepCopyWithVisited` methods. Generated code depends on this module's `deepcopy` package. Run `go mod tidy` to resolve dependencies, then `go test ./...` to validate.

## Drawbacks and Limitations

- **Not a complete deep copy of arbitrary objects:** dynamic interface values, generic parameters without copy methods, and external types without copy methods may retain shared internal data.
- **Runtime resources are not duplicated:** functions, channels, and `unsafe.Pointer` retain their identity. Resources such as locks are outside the safe-copy guarantees.
- **Reference preservation has boundaries:** map keys retain their original values. Overlapping slice backing arrays and pointer-free cycles involving maps or slices are not guaranteed to be preserved.
- **Syntax and build constraints:** dot imports, cgo types, generic aliases, and external type redefinitions that cannot be expanded are unsupported. Generation uses the current build environment; there is no custom build-tags option.
- **Generated files require maintenance:** regenerate after type or target build-environment changes, then compile to verify. Do not edit `deepcopy.dc.go` by hand.

See [SUPPORT.md](SUPPORT.md) (Chinese) for detailed support boundaries and testing notes.

## License

MIT
