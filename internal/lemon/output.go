package lemon

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func lineDirective(line int, filename string) string {
	return fmt.Sprintf("#line %d \"%s\"\n", line, strings.ReplaceAll(filename, "\\", "\\\\"))
}

func (g *Grammar) reprint(w io.Writer) {
	fmt.Fprintf(w, "// Reprint of input file \"%s\".\n// Symbols:\n", g.Filename)
	maxlen := 10
	for _, s := range g.Symbols[:g.NSymbol] {
		if len(s.Name) > maxlen {
			maxlen = len(s.Name)
		}
	}
	columns := 76 / (maxlen + 5)
	if columns < 1 {
		columns = 1
	}
	skip := (g.NSymbol + columns - 1) / columns
	for i := 0; i < skip; i++ {
		fmt.Fprint(w, "//")
		for j := i; j < g.NSymbol; j += skip {
			fmt.Fprintf(w, " %3d %-*s", j, maxlen, g.Symbols[j].Name)
		}
		fmt.Fprintln(w)
	}
	for _, r := range g.Rules {
		fmt.Fprintf(w, "%s ::=", r.LHS.Name)
		for _, s := range r.RHS {
			if s.Type == Multiterminal {
				for i, sub := range s.Subsymbols {
					if i == 0 {
						fmt.Fprint(w, " ")
					} else {
						fmt.Fprint(w, "|")
					}
					fmt.Fprint(w, sub.Name)
				}
			} else {
				fmt.Fprintf(w, " %s", s.Name)
			}
		}
		fmt.Fprint(w, ".")
		if r.PrecSymbol != nil {
			fmt.Fprintf(w, " [%s]", r.PrecSymbol.Name)
		}
		fmt.Fprintln(w)
	}
}

// Snapshot is a deterministic inspection format, not an upstream Lemon option.
// Hex text preserves every byte, including newlines and absent vs empty values.
func (g *Grammar) Snapshot(w io.Writer) {
	h := func(s string) string { return hex.EncodeToString([]byte(s)) }
	text := func(s *string) string {
		if s == nil {
			return "-"
		}
		return h(*s)
	}
	sym := func(s *Symbol) string {
		if s == nil {
			return "-"
		}
		return h(s.Name)
	}
	subs := func(s *Symbol) string {
		a := make([]string, len(s.Subsymbols))
		for i, v := range s.Subsymbols {
			a[i] = h(v.Name)
		}
		return strings.Join(a, ",")
	}
	fmt.Fprintf(w, "grammar\t%d\t%d\t%d\t%d\t%s\t%d\n", g.NSymbol, g.NTerminal, g.NRuleWithAction, boolInt(g.HasFallback), sym(g.Wildcard), g.StartRule.Index)
	for _, d := range []struct {
		name  string
		value *string
	}{
		{"name", g.Name}, {"include", g.Include}, {"code", g.ExtraCode}, {"token_destructor", g.TokenDestructor}, {"default_destructor", g.DefaultDestructor}, {"token_prefix", g.TokenPrefix}, {"syntax_error", g.SyntaxError}, {"parse_accept", g.ParseAccept}, {"parse_failure", g.ParseFailure}, {"stack_overflow", g.StackOverflow}, {"extra_argument", g.ExtraArgument}, {"extra_context", g.ExtraContext}, {"token_type", g.TokenType}, {"default_type", g.DefaultType}, {"realloc", g.Realloc}, {"free", g.Free}, {"stack_size", g.StackSize}, {"start_symbol", g.StartSymbol},
	} {
		fmt.Fprintf(w, "decl\t%s\t%s\n", d.name, text(d.value))
	}
	for _, s := range g.Symbols {
		var rules []string
		for r := s.Rule; r != nil; r = r.NextLHS {
			rules = append(rules, strconv.Itoa(r.Index))
		}
		fmt.Fprintf(w, "symbol\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%d\t%s\t%d\t%s\t%s\n", h(s.Name), s.Index, s.Type, s.UseCount, s.Prec, s.Assoc, sym(s.Fallback), text(s.Destructor), s.DestLine, text(s.Datatype), boolInt(s.Content), subs(s), strings.Join(rules, ","))
	}
	for _, r := range g.Rules {
		fmt.Fprintf(w, "rule\t%d\t%d\t%s\t%s\t%d\t%d\t%s\t%d\t%d\t%s\t%d\n", r.Index, r.IRule, h(r.LHS.Name), text(r.LHSAlias), r.RuleLine, r.Line, text(r.Code), boolInt(r.NoCode), boolInt(r.NeverReduce), sym(r.PrecSymbol), len(r.RHS))
		for i, s := range r.RHS {
			fmt.Fprintf(w, "rhs\t%s\t%d\t%s\t%s\n", h(s.Name), s.Type, text(r.RHSAlias[i]), subs(s))
		}
	}
	for i, d := range g.defines {
		fmt.Fprintf(w, "define\t%s\t%d\n", h(d), boolInt(g.defineUsed[i]))
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
