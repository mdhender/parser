package lemon

import (
	"fmt"
	"io"
)

// ParseGrammar exposes only Lemon's preprocessing, scanner, and grammar parser.
// Actions use Go literal scanning. Rules and symbols retain source order; no
// LR preparation, state construction, or table generation is performed.
// The returned grammar belongs to the caller, including all symbol identities.
func ParseGrammar(filename string, input []byte, defines []string, stderr io.Writer) (*Grammar, error) {
	g := &Grammar{Filename: filename, byName: make(map[string]*Symbol),
		Declarations: make(map[string]int), defines: append([]string(nil), defines...),
		defineUsed: make([]bool, len(defines)), noLineNumbers: true, goTarget: true, stderr: stderr}
	g.symbol("$")
	z, ok := g.preprocess(input)
	if !ok {
		return nil, fmt.Errorf("grammar preprocessing failed")
	}
	g.scan(z)
	if g.errors != 0 {
		return nil, fmt.Errorf("grammar parsing failed")
	}
	if len(g.Rules) == 0 {
		return nil, fmt.Errorf("%s: empty grammar", filename)
	}
	for i, s := range g.Symbols {
		s.Index = i
	}
	g.StartRule = g.Rules[0]
	g.ErrorSymbol = g.byName["error"]
	return g, nil
}
