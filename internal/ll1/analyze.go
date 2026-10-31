// Package ll1 analyzes Lemon grammars for predictive, single-token parsing.
package ll1

import (
	"fmt"
	"io"
	"strings"

	"github.com/mdhender/parser/internal/lemon"
)

// Set is a set of terminal symbols. Epsilon is represented by Nullable, not by
// a synthetic terminal. The original grammar's $ symbol represents EOF.
type Set map[*lemon.Symbol]bool

// Analysis owns LL(1) results without changing the parsed grammar. A table with
// errors is diagnostic only: conflicting entries retain the first production.
type Analysis struct {
	Grammar  *lemon.Grammar
	Start    *lemon.Symbol
	Nullable map[*lemon.Symbol]bool
	First    map[*lemon.Symbol]Set
	Follow   map[*lemon.Symbol]Set
	Select   map[*lemon.Rule]Set
	Table    map[*lemon.Symbol]map[*lemon.Symbol]*lemon.Rule
	Errors   []string
	Warnings []string
}

// Analyze computes the least fixed points for nullable, FIRST, FOLLOW and
// SELECT, and checks grammar properties needed by an LL(1) parser. It does not
// prepare the grammar or invoke any Lemon LR construction routines.
func Analyze(g *lemon.Grammar) *Analysis {
	a := &Analysis{
		Grammar: g, Nullable: make(map[*lemon.Symbol]bool),
		First: make(map[*lemon.Symbol]Set), Follow: make(map[*lemon.Symbol]Set),
		Select: make(map[*lemon.Rule]Set),
		Table:  make(map[*lemon.Symbol]map[*lemon.Symbol]*lemon.Rule),
	}
	if g == nil {
		a.Errors = append(a.Errors, "no grammar to analyze")
		return a
	}
	definitions := make(map[*lemon.Symbol][]*lemon.Rule)
	locations := make(map[*lemon.Symbol]int)
	for _, r := range g.Rules {
		definitions[r.LHS] = append(definitions[r.LHS], r)
		for _, s := range append([]*lemon.Symbol{r.LHS}, r.RHS...) {
			if _, ok := locations[s]; !ok {
				locations[s] = r.RuleLine
			}
		}
	}
	var eof *lemon.Symbol
	for _, s := range g.Symbols {
		a.First[s], a.Follow[s] = make(Set), make(Set)
		// The raw frontend classifies $ by its spelling, unlike ordinary
		// uppercase terminals. It is nevertheless the EOF terminal for LL(1).
		if s.Type == lemon.Terminal || s.Name == "$" {
			a.First[s][s] = true
		}
		if s.Name == "$" {
			eof = s
		}
		if s.Type == lemon.Nonterminal && s.Name != "$" {
			a.Table[s] = make(map[*lemon.Symbol]*lemon.Rule)
		}
	}
	if len(g.Rules) == 0 {
		a.problem(0, "grammar has no productions; add a start production")
	} else {
		a.Start = g.Rules[0].LHS
	}
	if g.StartSymbol != nil {
		a.Start = nil
		for _, s := range g.Symbols {
			if s.Name == *g.StartSymbol && s.Type == lemon.Nonterminal && len(definitions[s]) != 0 {
				a.Start = s
				break
			}
		}
		if a.Start == nil {
			a.problem(0, "%%start_symbol %q has no productions; choose a defined nonterminal", *g.StartSymbol)
		}
	}
	if eof == nil {
		a.problem(0, "grammar is missing the $ end-of-input symbol")
	}
	if g.HasFallback {
		a.problem(0, "%%fallback is unsupported in LL(1); use explicit token alternatives")
	}
	if g.Wildcard != nil {
		a.problem(locations[g.Wildcard], "%%wildcard is unsupported in LL(1); enumerate accepted tokens")
	}
	for _, s := range g.Symbols {
		line := locations[s]
		if s.Fallback != nil && !g.HasFallback {
			a.problem(line, "fallback for %s is unsupported in LL(1); use explicit token alternatives", s.Name)
		}
		if s.Name == "error" || s == g.ErrorSymbol {
			a.problem(line, "error recovery symbol %s is unsupported in LL(1); remove error productions", s.Name)
		}
		if s.Type == lemon.Multiterminal {
			a.problem(line, "multiterminal %s is unsupported in LL(1); split it into explicit productions", s.Name)
		}
		if s.Prec >= 0 {
			a.problem(line, "precedence for %s is unsupported in LL(1); precedence does not resolve LL(1) conflicts; rewrite the grammar", s.Name)
		}
		if s.Type == lemon.Nonterminal && len(definitions[s]) == 0 && s.Name != "error" && s.Name != "$" {
			a.problem(line, "undefined nonterminal %s; add a production or correct its name", s.Name)
		}
	}
	for _, r := range g.Rules {
		if r.PrecSymbol != nil {
			a.problem(r.RuleLine, "precedence override on %s is unsupported in LL(1); precedence does not resolve LL(1) conflicts; rewrite the grammar", production(r))
		}
		if r.NeverReduce {
			a.problem(r.RuleLine, "NEVER-REDUCE on %s is unsupported in LL(1); remove this LR-only annotation", production(r))
		}
		// Inline multiterminals are not necessarily in Grammar.Symbols.
		for _, s := range r.RHS {
			if s.Type == lemon.Multiterminal {
				a.problem(r.RuleLine, "multiterminal in %s is unsupported in LL(1); split it into explicit productions", production(r))
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, r := range g.Rules {
			first, nullable := a.sequence(r.RHS)
			if merge(a.First[r.LHS], first) {
				changed = true
			}
			if nullable && !a.Nullable[r.LHS] {
				a.Nullable[r.LHS], changed = true, true
			}
		}
	}
	if a.Start != nil && eof != nil {
		a.Follow[a.Start][eof] = true
	}
	for changed := true; changed; {
		changed = false
		for _, r := range g.Rules {
			for i, s := range r.RHS {
				if s.Type != lemon.Nonterminal {
					continue
				}
				first, nullable := a.sequence(r.RHS[i+1:])
				if merge(a.Follow[s], first) {
					changed = true
				}
				if nullable && merge(a.Follow[s], a.Follow[r.LHS]) {
					changed = true
				}
			}
		}
	}
	for _, r := range g.Rules {
		selectSet, nullable := a.sequence(r.RHS)
		if nullable {
			merge(selectSet, a.Follow[r.LHS])
		}
		a.Select[r] = selectSet
		for _, token := range g.Symbols {
			// A rejected token class can still occur as an LHS in the
			// shared frontend. Keep analyzing it without creating a row.
			if a.Table[r.LHS] != nil && selectSet[token] && a.Table[r.LHS][token] == nil {
				a.Table[r.LHS][token] = r
			}
		}
	}
	// Compare every pair, not just each alternative against the table winner.
	for i, r := range g.Rules {
		for _, other := range g.Rules[i+1:] {
			if other.LHS != r.LHS {
				continue
			}
			firstR, _ := a.sequence(r.RHS)
			firstOther, _ := a.sequence(other.RHS)
			ff, follow := make(Set), make(Set)
			for _, token := range g.Symbols {
				if a.Select[r][token] && a.Select[other][token] {
					if firstR[token] && firstOther[token] {
						ff[token] = true
					} else {
						follow[token] = true
					}
				}
			}
			for _, conflict := range []struct {
				kind, advice string
				set          Set
			}{
				{"FIRST/FIRST", "left-factor the common prefix or rewrite the alternatives", ff},
				{"FIRST/FOLLOW", "rewrite nullable alternatives or their following context", follow},
			} {
				if len(conflict.set) != 0 {
					a.problem(r.RuleLine, "%s conflict between %s and %s:%d: %s; overlap %s; %s", conflict.kind, production(r), g.Filename, other.RuleLine, production(other), a.setText(conflict.set), conflict.advice)
				}
			}
		}
	}
	a.checkLeftRecursion()
	a.checkUseful(definitions, locations)
	return a
}

func merge(dst, src Set) bool {
	changed := false
	for s, present := range src {
		if present && !dst[s] {
			dst[s], changed = true, true
		}
	}
	return changed
}

func (a *Analysis) sequence(symbols []*lemon.Symbol) (Set, bool) {
	first := make(Set)
	for _, s := range symbols {
		merge(first, a.First[s])
		if !a.Nullable[s] {
			return first, false
		}
	}
	return first, true
}

func (a *Analysis) problem(line int, format string, args ...any) {
	a.Errors = append(a.Errors, fmt.Sprintf("%s:%d: %s", a.Grammar.Filename, line, fmt.Sprintf(format, args...)))
}

func production(r *lemon.Rule) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s ::=", r.LHS.Name)
	for _, s := range r.RHS {
		fmt.Fprintf(&b, " %s", s.Name)
	}
	b.WriteString(" .")
	return b.String()
}

func (a *Analysis) setText(set Set) string {
	names := make([]string, 0, len(set))
	for _, s := range a.Grammar.Symbols {
		if set[s] {
			names = append(names, s.Name)
		}
	}
	return "{" + strings.Join(names, ", ") + "}"
}

// A left-corner edge is possible only through a nullable RHS prefix. A cycle
// in this graph is exactly direct or indirect left recursion, including cycles
// whose symbols cannot derive any terminal string.
func (a *Analysis) checkLeftRecursion() {
	type edge struct {
		to   *lemon.Symbol
		rule *lemon.Rule
	}
	edges := make(map[*lemon.Symbol][]edge)
	for _, r := range a.Grammar.Rules {
		for _, s := range r.RHS {
			if s.Type == lemon.Nonterminal {
				edges[r.LHS] = append(edges[r.LHS], edge{s, r})
			}
			if !a.Nullable[s] {
				break
			}
		}
	}
	for _, root := range a.Grammar.Symbols {
		visited := make(map[*lemon.Symbol]bool)
		var path []edge
		var visit func(*lemon.Symbol) bool
		visit = func(s *lemon.Symbol) bool {
			visited[s] = true
			for _, e := range edges[s] {
				path = append(path, e)
				if e.to == root || (!visited[e.to] && visit(e.to)) {
					return true
				}
				path = path[:len(path)-1]
			}
			return false
		}
		if visit(root) {
			parts := make([]string, 0, len(path))
			for _, e := range path {
				parts = append(parts, fmt.Sprintf("%s:%d: %s", a.Grammar.Filename, e.rule.RuleLine, production(e.rule)))
			}
			a.problem(path[0].rule.RuleLine, "left recursion for %s through nullable prefixes: %s; eliminate left recursion before LL(1) parsing", root.Name, strings.Join(parts, " -> "))
		}
	}
}

func (a *Analysis) checkUseful(definitions map[*lemon.Symbol][]*lemon.Rule, locations map[*lemon.Symbol]int) {
	productive := make(map[*lemon.Symbol]bool)
	for _, s := range a.Grammar.Symbols {
		productive[s] = s.Type == lemon.Terminal || s.Name == "$"
	}
	for changed := true; changed; {
		changed = false
		for _, r := range a.Grammar.Rules {
			all := true
			for _, s := range r.RHS {
				all = all && productive[s]
			}
			if all && !productive[r.LHS] {
				productive[r.LHS], changed = true, true
			}
		}
	}
	for _, s := range a.Grammar.Symbols {
		if s.Type == lemon.Nonterminal && len(definitions[s]) != 0 && !productive[s] {
			a.problem(locations[s], "nonproductive nonterminal %s cannot derive a terminal string; add a terminating alternative", s.Name)
		}
	}
	reachable := make(map[*lemon.Symbol]bool)
	if a.Start != nil {
		reachable[a.Start] = true
	}
	for changed := true; changed; {
		changed = false
		for _, r := range a.Grammar.Rules {
			if reachable[r.LHS] {
				for _, s := range r.RHS {
					if !reachable[s] {
						reachable[s], changed = true, true
					}
				}
			}
		}
	}
	for _, r := range a.Grammar.Rules {
		if !reachable[r.LHS] {
			a.Warnings = append(a.Warnings, fmt.Sprintf("%s:%d: unreachable production %s; remove it or reference it from the start symbol", a.Grammar.Filename, r.RuleLine, production(r)))
		}
	}
}

// Explain prints source-order symbols, productions, and table entries. Map
// iteration is never used to choose presentation order.
func (a *Analysis) Explain(w io.Writer) {
	if a.Start != nil {
		fmt.Fprintf(w, "Start: %s\n", a.Start.Name)
	} else {
		fmt.Fprintln(w, "Start: (none)")
	}
	if a.Grammar != nil {
		fmt.Fprintln(w, "Symbols:")
		for _, s := range a.Grammar.Symbols {
			fmt.Fprintf(w, "  %s: nullable=%t FIRST=%s FOLLOW=%s\n", s.Name, a.Nullable[s], a.setText(a.First[s]), a.setText(a.Follow[s]))
		}
		fmt.Fprintln(w, "Productions:")
		for _, r := range a.Grammar.Rules {
			fmt.Fprintf(w, "  %s:%d: %s SELECT=%s\n", a.Grammar.Filename, r.RuleLine, production(r), a.setText(a.Select[r]))
		}
		fmt.Fprintln(w, "Table:")
		for _, s := range a.Grammar.Symbols {
			for _, token := range a.Grammar.Symbols {
				if r := a.Table[s][token]; r != nil {
					fmt.Fprintf(w, "  [%s, %s] = %s:%d: %s\n", s.Name, token.Name, a.Grammar.Filename, r.RuleLine, production(r))
				}
			}
		}
	}
	for _, err := range a.Errors {
		fmt.Fprintf(w, "Error: %s\n", err)
	}
	for _, warning := range a.Warnings {
		fmt.Fprintf(w, "Warning: %s\n", warning)
	}
}
