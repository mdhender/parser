# Build and install Lemon

Use this guide to install the Go generator from source. You need Git and Go
**1.27.3**, as declared in [`go.mod`](../../go.mod). A C compiler is not needed
to build or run the generator; it is needed to compile the generated C parser.

## Build from a checkout

```sh
git clone https://github.com/mdhender/parser.git
cd parser
CGO_ENABLED=0 go build -o lemon ./cmd/lemon
./lemon -x
```

Expected output: `Lemon version 1.0`. This is the upstream Lemon version string,
not the Go port release or the SQLite revision. The command package uses only
the standard library and needs neither cgo nor downloaded modules to build.

To install the command into Go's binary directory instead:

```sh
CGO_ENABLED=0 go install ./cmd/lemon
```

The executable is placed in `GOBIN`, or in the first `GOPATH` entry's `bin`
directory when `GOBIN` is unset (normally `$HOME/go/bin`). Add that directory
to your shell's PATH if you want to invoke `lemon` by name. You do not need to
copy `lempar.c`: the default template is embedded.

## Platform boundary

The recorded generation/runtime compatibility results are for Linux x86-64.
The generator uses portable Go code, but native macOS/Windows compatibility is
not established by those results. These examples use a POSIX shell; the full
maintainer suite also has GNU-tool and Linux ELF requirements. See
[compatibility](../reference/compatibility.md) rather than assuming a successful
cross-build establishes byte-for-byte output compatibility.

Continue with the [first-parser tutorial](../tutorials/first-parser.md) or
[generate from an existing grammar](generate-parser.md). Build/test contributors
should use the separate [development instructions](../development/README.md),
including the root version API's module dependency.
