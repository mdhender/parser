// Package ll1 analyzes Lemon-format grammars and emits predictive Go parsers.
package ll1

import (
	"flag"
	"fmt"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mdhender/parser/internal/lemon"
)

// Run implements glime. Flags precede the subcommand; the subcommand is required.
func Run(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.SetOutput(stderr)
	strategy := f.String("strategy", "stack", "parser strategy (recursive is not implemented)")
	pkg := f.String("package", "parser", "generated Go package")
	dir := f.String("d", "", "output directory (must exist)")
	var defines []string
	f.Func("D", "define a grammar preprocessor name (repeatable)", func(s string) error {
		defines = append(defines, strings.SplitN(s, "=", 2)[0])
		return nil
	})
	f.Usage = func() {
		fmt.Fprintln(stderr, "Usage: glime [flags] check|explain|generate grammar.y")
		f.PrintDefaults()
	}
	if err := f.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}
	if *strategy != "stack" {
		if *strategy == "recursive" {
			fmt.Fprintln(stderr, "strategy recursive is not implemented")
		} else {
			fmt.Fprintf(stderr, "unknown strategy %q\n", *strategy)
		}
		return 1
	}
	if f.NArg() != 2 || (f.Arg(0) != "check" && f.Arg(0) != "explain" && f.Arg(0) != "generate") {
		f.Usage()
		return 1
	}
	if !token.IsIdentifier(*pkg) || *pkg == "_" {
		fmt.Fprintln(stderr, "invalid Go package name")
		return 1
	}
	input, err := os.ReadFile(f.Arg(1))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	g, err := lemon.ParseGrammar(f.Arg(1), input, defines, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	a := Analyze(g)
	// Reject declarations even when empty, rather than silently ignoring them.
	// Iterate a fixed inventory for reproducible diagnostics.
	for _, key := range []string{"left", "right", "nonassoc", "fallback", "wildcard", "token_class", "destructor", "token_destructor", "default_destructor", "extra_argument", "extra_context", "stack_size", "stack_overflow", "realloc", "free"} {
		if line, exists := g.Declarations[key]; exists {
			a.Errors = append(a.Errors, fmt.Sprintf("%s:%d: %%%s is not supported by glime", g.Filename, line, key))
		}
	}
	// Validate Go declarations/actions for check as well as generate.
	var source []byte
	if len(a.Errors) == 0 {
		source, err = a.Emit(*pkg)
		if err != nil {
			a.Errors = append(a.Errors, err.Error())
		}
	}
	if f.Arg(0) == "explain" {
		a.Explain(stdout)
	}
	for _, warning := range a.Warnings {
		fmt.Fprintln(stderr, "warning:", warning)
	}
	for _, diagnostic := range a.Errors {
		fmt.Fprintln(stderr, diagnostic)
	}
	if len(a.Errors) != 0 {
		return 1
	}
	if f.Arg(0) == "generate" {
		base := filepath.Base(g.Filename)
		base = strings.TrimSuffix(base, filepath.Ext(base)) + ".go"
		outdir := *dir
		if outdir == "" {
			outdir = filepath.Dir(g.Filename)
		}
		if err := os.WriteFile(filepath.Join(outdir, base), source, 0644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	return 0
}
