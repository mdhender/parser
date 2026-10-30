package lemon

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// reportRulePrint translates RulePrint. A negative cursor omits the dot and
// produces the same text as rule_print and writeRuleText in the C reference.
func reportRulePrint(w io.Writer, r *Rule, cursor int) {
	fmt.Fprintf(w, "%s ::=", r.LHS.Name)
	for i := 0; i <= len(r.RHS); i++ {
		if i == cursor {
			fmt.Fprint(w, " *")
		}
		if i == len(r.RHS) {
			break
		}
		s := r.RHS[i]
		if s.Type == Multiterminal {
			fmt.Fprintf(w, " %s", s.Subsymbols[0].Name)
			for _, sub := range s.Subsymbols[1:] {
				fmt.Fprintf(w, "|%s", sub.Name)
			}
		} else {
			fmt.Fprintf(w, " %s", s.Name)
		}
	}
}

// reportPrintAction preserves PrintAction's spacing, including the different
// indentation of a reduce action dropped by precedence.
func reportPrintAction(w io.Writer, a *Action, indent int, precedence bool) bool {
	switch a.Type {
	case Shift:
		fmt.Fprintf(w, "%*s shift        %-7d", indent, a.Symbol.Name, a.State.Number)
	case Reduce:
		fmt.Fprintf(w, "%*s reduce       %-7d", indent, a.Symbol.Name, a.Rule.IRule)
		reportRulePrint(w, a.Rule, -1)
	case ShiftReduce:
		fmt.Fprintf(w, "%*s shift-reduce %-7d", indent, a.Symbol.Name, a.Rule.IRule)
		reportRulePrint(w, a.Rule, -1)
	case Accept:
		fmt.Fprintf(w, "%*s accept", indent, a.Symbol.Name)
	case Error:
		fmt.Fprintf(w, "%*s error", indent, a.Symbol.Name)
	case SRConflict, RRConflict:
		fmt.Fprintf(w, "%*s reduce       %-7d ** Parsing conflict **", indent, a.Symbol.Name, a.Rule.IRule)
	case SSConflict:
		fmt.Fprintf(w, "%*s shift        %-7d ** Parsing conflict **", indent, a.Symbol.Name, a.State.Number)
	case SHResolved:
		if !precedence {
			return false
		}
		fmt.Fprintf(w, "%*s shift        %-7d -- dropped by precedence", indent, a.Symbol.Name, a.State.Number)
	case RDResolved:
		if !precedence {
			return false
		}
		fmt.Fprintf(w, "%*s reduce %-7d -- dropped by precedence", indent, a.Symbol.Name, a.Rule.IRule)
	case NotUsed:
		return false
	}
	if a.Opt != nil {
		fmt.Fprintf(w, "  /* because %s==%s */", a.Symbol.Name, a.Opt.Name)
	}
	return true
}

func (g *Grammar) reportOutput(o options) {
	f := g.openOutput(".out", o)
	if f == nil {
		return
	}
	defer f.Close()
	for _, s := range g.States[:g.NXState] {
		fmt.Fprintf(f, "State %d:\n", s.Number)
		configs := s.Configs
		if o.flags['b'] {
			configs = s.Basis
		}
		for _, c := range configs {
			if c.Dot == len(c.Rule.RHS) {
				fmt.Fprintf(f, "    %5s ", fmt.Sprintf("(%d)", c.Rule.IRule))
			} else {
				fmt.Fprint(f, "          ")
			}
			reportRulePrint(f, c.Rule, c.Dot)
			fmt.Fprintln(f)
		}
		fmt.Fprintln(f)
		for _, a := range s.Actions {
			if reportPrintAction(f, a, 30, o.flags['p']) {
				fmt.Fprintln(f)
			}
		}
		fmt.Fprintln(f)
	}
	fmt.Fprint(f, "----------------------------------------------------\nSymbols:\nThe first-set of non-terminals is shown after the name.\n\n")
	for i, s := range g.Symbols[:g.NSymbol] {
		fmt.Fprintf(f, "  %3d: %s", i, s.Name)
		if s.Type == Nonterminal {
			fmt.Fprint(f, ":")
			if s.Nullable {
				fmt.Fprint(f, " <lambda>")
			}
			for j := 0; j < g.NTerminal; j++ {
				if s.First != nil && s.First[j] != 0 {
					fmt.Fprintf(f, " %s", g.Symbols[j].Name)
				}
			}
		}
		if s.Prec >= 0 {
			fmt.Fprintf(f, " (precedence=%d)", s.Prec)
		}
		fmt.Fprintln(f)
	}
	fmt.Fprint(f, "----------------------------------------------------\nSyntax-only Symbols:\nThe following symbols never carry semantic content.\n\n")
	n := 0
	for _, s := range g.Symbols[:g.NSymbol] {
		if s.Content {
			continue
		}
		width := len(s.Name)
		if n > 0 && n+width > 75 {
			fmt.Fprintln(f)
			n = 0
		}
		if n > 0 {
			fmt.Fprint(f, " ")
			n++
		}
		fmt.Fprint(f, s.Name)
		n += width
	}
	if n > 0 {
		fmt.Fprintln(f)
	}
	fmt.Fprint(f, "----------------------------------------------------\nRules:\n")
	for _, r := range g.Rules {
		fmt.Fprintf(f, "%4d: ", r.IRule)
		reportRulePrint(f, r, -1)
		fmt.Fprint(f, ".")
		if r.PrecSymbol != nil {
			fmt.Fprintf(f, " [%s precedence=%d]", r.PrecSymbol.Name, r.PrecSymbol.Prec)
		}
		fmt.Fprintln(f)
	}
}

func (g *Grammar) reportHeader(o options) {
	prefix := ""
	if g.TokenPrefix != nil {
		prefix = *g.TokenPrefix
	}
	pattern := func(i int) string {
		return fmt.Sprintf("#define %s%-30s %3d\n", prefix, g.Symbols[i].Name, i)
	}
	// Reading via file_open also updates outname, but does not diagnose a
	// missing header. Only the subsequent write reports an open failure.
	g.outname = g.fileName(".h", o)
	if in, err := os.Open(g.outname); err == nil {
		reader := bufio.NewReader(in)
		i := 1
		for ; i < g.NTerminal; i++ {
			// fgets(line, LINESIZE, in) reads at most 999 bytes.
			line := make([]byte, 0, 999)
			for len(line) < 999 {
				c, err := reader.ReadByte()
				if err != nil {
					break
				}
				line = append(line, c)
				if c == '\n' {
					break
				}
			}
			if len(line) == 0 || string(line) != pattern(i) {
				break
			}
		}
		_, err := reader.ReadByte()
		in.Close()
		if i == g.NTerminal && err == io.EOF {
			return // Unchanged headers retain their timestamps.
		}
	}
	if out := g.openOutput(".h", o); out != nil {
		for i := 1; i < g.NTerminal; i++ {
			fmt.Fprint(out, pattern(i))
		}
		out.Close()
	}
}

// reportSQL is the SQL block of ReportTable. Opening this file intentionally
// leaves outname naming the SQL file, just as file_open does in the reference.
func (g *Grammar) reportSQL(o options) {
	f := g.openOutput(".sql", o)
	if f == nil {
		return
	}
	defer f.Close()
	fmt.Fprint(f, "BEGIN;\nCREATE TABLE symbol(\n  id INTEGER PRIMARY KEY,\n  name TEXT NOT NULL,\n  isTerminal BOOLEAN NOT NULL,\n  fallback INTEGER REFERENCES symbol DEFERRABLE INITIALLY DEFERRED\n);\n")
	for i, s := range g.Symbols[:g.NSymbol] {
		terminal := "FALSE"
		if i < g.NTerminal {
			terminal = "TRUE"
		}
		// C writes names and rule text literally: it performs no SQL escaping.
		fmt.Fprintf(f, "INSERT INTO symbol(id,name,isTerminal,fallback)VALUES(%d,'%s',%s", i, s.Name, terminal)
		if s.Fallback != nil {
			fmt.Fprintf(f, ",%d);\n", s.Fallback.Index)
		} else {
			fmt.Fprint(f, ",NULL);\n")
		}
	}
	fmt.Fprint(f, "CREATE TABLE rule(\n  ruleid INTEGER PRIMARY KEY,\n  lhs INTEGER REFERENCES symbol(id),\n  txt TEXT\n);\nCREATE TABLE rulerhs(\n  ruleid INTEGER REFERENCES rule(ruleid),\n  pos INTEGER,\n  sym INTEGER REFERENCES symbol(id)\n);\n")
	for i, r := range g.Rules {
		fmt.Fprintf(f, "INSERT INTO rule(ruleid,lhs,txt)VALUES(%d,%d,'", r.IRule, r.LHS.Index)
		reportRulePrint(f, r, -1)
		fmt.Fprint(f, "');\n")
		for j, s := range r.RHS {
			if s.Type != Multiterminal {
				fmt.Fprintf(f, "INSERT INTO rulerhs(ruleid,pos,sym)VALUES(%d,%d,%d);\n", i, j, s.Index)
			} else {
				for _, sub := range s.Subsymbols {
					fmt.Fprintf(f, "INSERT INTO rulerhs(ruleid,pos,sym)VALUES(%d,%d,%d);\n", i, j, sub.Index)
				}
			}
		}
	}
	fmt.Fprint(f, "COMMIT;\n")
}
