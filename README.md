# parser

A faithful Go port of SQLite's Lemon parser generator, currently at
**Milestone A: reproducible C reference baseline**. No Go generator has been
implemented yet. The eventual Go executable will generate C parsers.

The reference is pinned to SQLite 3.50.4. Pristine sources and golden outputs
are committed; tests do not download anything or update fixtures.

## Validate

Offline fixture-integrity checks (Go version from `go.mod`; no C compiler):

```sh
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go test ./...
GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 go vet ./...
```

Independent C reference rebuilds and byte-for-byte golden comparisons
(Bash, `cc`, GNU coreutils and diffutils):

```sh
scripts/test-reference.sh
```

See [reference provenance and reproduction](docs/REFERENCE.md),
[architecture](docs/ARCHITECTURE.md), [porting log](docs/PORTING.md), and
[compatibility scope](docs/COMPATIBILITY.md). Work stops at Milestone A pending
authorization for translation.
