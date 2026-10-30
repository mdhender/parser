package reference

import _ "embed"

// ParserTemplate is the pristine SQLite parser driver pinned in SHA256SUMS.
// Embedding the source directly keeps the standalone default and oracle identical.
//
//go:embed sqlite/tool/lempar.c
var ParserTemplate string
