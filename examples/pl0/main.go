// Command pl0 runs the same PL/0 interpreter with either Go parser frontend.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mdhender/parser/examples/pl0/lalr"
	"github.com/mdhender/parser/examples/pl0/lang"
	"github.com/mdhender/parser/examples/pl0/ll1"
)

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("pl0", flag.ContinueOnError)
	flags.SetOutput(stderr)
	parser := flags.String("parser", "glcc", "parser frontend: glcc or glime")
	flags.Usage = func() { fmt.Fprintln(stderr, "Usage: pl0 [-parser glcc|glime] program.pl0") }
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 1
	}
	parse := lalr.Parse
	if *parser == "glime" {
		parse = ll1.Parse
	} else if *parser != "glcc" {
		fmt.Fprintf(stderr, "unknown parser %q; choose glcc or glime\n", *parser)
		return 1
	}
	filename := flags.Arg(0)
	source, err := os.ReadFile(filename)
	if err == nil {
		var block *lang.Block
		block, err = parse(string(source))
		if err == nil {
			err = lang.Execute(block, stdout)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", filename, err)
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
