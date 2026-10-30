# Porting log

## Milestone A — Pinned C reference baseline

- Reference: SQLite 3.50.4, Fossil check-in
  `4d8adfb30e03f9cf27f800a2c1ba3c48fb4ca1b08b0f5ed59a4d5ecbf45e20a3`.
- Sources: pristine `lemon.c`, `lempar.c`, `parse.y`, and supporting provenance
  files under `reference/sqlite/`; SHA-256 manifest under `reference/`.
- C functions translated: **none**. There is no Go generator yet.
- Go files: `reference/reference_test.go` checks fixture integrity offline.
- Tooling: `scripts/generate-reference.sh` builds/runs the oracle;
  `scripts/compare-artifacts.sh` compares files and process results;
  `scripts/test-reference.sh` independently rebuilds/reruns twice and compares
  against the goldens and between runs.
- Configuration and artifact hashes: [REFERENCE.md](REFERENCE.md).
- Initial subsystem analysis: [ARCHITECTURE.md](ARCHITECTURE.md).
- Compatibility scope and unimplemented features: [COMPATIBILITY.md](COMPATIBILITY.md).
- Intentional deviations from upstream: none in vendored source or generated
  outputs. The scripts isolate directories, fix locale/time zone, and capture
  diagnostics/metadata without changing the reference code.
- The baseline includes `parse.sql` because upstream's `-S` enables it, in
  addition to `parse.c`, `parse.h`, and `parse.out`.
- Validation: `scripts/test-reference.sh` passed, reproducing all four generated
  files plus stdout, stderr, and exit status byte for byte in two independent
  builds/runs. All generated files also passed `diff -bw`.
- Offline Go validation: `go test -count=3 ./...` and `go vet ./...` passed with
  `GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0`; the repeated test also used a
  nonexistent `CC` to check that ordinary tests do not need a C compiler.
- Negative validation in disposable copies: changed source/golden hashes,
  each changed output/process result, missing artifacts, and whitespace-only
  output changes were rejected. Missing compiler checks failed explicitly and
  retained their work directory. Generation/comparison with spaces in the
  output path passed; reusing an existing output directory was rejected.
- Unresolved work: Go translation and C-versus-Go differential validation have
  not started. No SQLite runtime/build validation is claimed.
- Next recommended task, **only after authorization**: Milestone B, starting
  with command-line processing, byte-oriented grammar scanning/preprocessing,
  symbol/rule representation, and small grammar fixtures. Compare observable
  preprocessed/reprinted grammar output with the pinned C oracle before moving
  to automaton construction.
