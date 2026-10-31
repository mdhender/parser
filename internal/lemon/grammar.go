// Package lemon implements SQLite's Lemon C parser generator. The authoritative
// source is reference/sqlite/tool/lemon.c.
package lemon

import (
	"fmt"
	"io"
	"sort"
)

type SymbolType int

const (
	Terminal SymbolType = iota
	Nonterminal
	Multiterminal
)

type Assoc int

const (
	Left Assoc = iota
	Right
	None
	Unknown
)

type Symbol struct {
	Name       string
	Index      int
	Type       SymbolType
	Rule       *Rule // Reverse source order, via NextLHS.
	Fallback   *Symbol
	Prec       int
	Assoc      Assoc
	UseCount   int
	Destructor *string
	DestLine   int
	Datatype   *string
	Content    bool
	Dtnum      int
	Subsymbols []*Symbol
	Nullable   bool
	First      symbolSet
}

type Rule struct {
	LHS                    *Symbol
	LHSAlias               *string
	RuleLine               int
	RHS                    []*Symbol
	RHSAlias               []*string
	Line                   int
	Code                   *string // nil differs from an explicitly empty action.
	PrecSymbol             *Symbol
	Index                  int // Original source identity, never renumbered.
	IRule                  int // Action-first generated rule number.
	NoCode                 bool
	NeverReduce            bool
	NextLHS                *Rule
	LHSStart               bool
	CanReduce              bool
	DoesReduce             bool
	CodePrefix, CodeSuffix *string
}

// Grammar owns all mutable state for one invocation. Maps are used only for
// lookup; ordered slices preserve the C associative arrays' insertion order.
type Grammar struct {
	Filename                                                             string
	Declarations                                                         map[string]int // Optional parse-only declaration source locations.
	Symbols                                                              []*Symbol
	Rules                                                                []*Rule
	StartRule                                                            *Rule
	States                                                               []*State
	NConflict                                                            int
	NXState                                                              int
	MinShiftReduce, ErrAction, AccAction, NoAction, MinReduce, MaxAction int
	ErrorSymbol, Wildcard                                                *Symbol
	NSymbol, NTerminal, NRuleWithAction                                  int
	HasFallback                                                          bool
	Name, Include, ExtraCode, TokenDestructor, DefaultDestructor         *string
	TokenPrefix, SyntaxError, ParseAccept, ParseFailure, StackOverflow   *string
	ExtraArgument, ExtraContext, TokenType, DefaultType                  *string
	Realloc, Free, StackSize, StartSymbol                                *string
	byName                                                               map[string]*Symbol
	defines                                                              []string
	defineUsed                                                           []bool
	noLineNumbers                                                        bool
	goTarget                                                             bool
	outname                                                              string
	tableSize, nActionTab, nLookaheadTab                                 int
	errors                                                               int
	stderr                                                               io.Writer
}

func (g *Grammar) symbol(name string) *Symbol {
	s := g.byName[name]
	if s == nil {
		s = &Symbol{Name: name, Prec: -1, Assoc: Unknown, Type: Nonterminal}
		if upper(name[0]) {
			s.Type = Terminal
		}
		g.byName[name] = s
		g.Symbols = append(g.Symbols, s)
	}
	s.UseCount++
	return s
}

func (g *Grammar) error(line int, format string, args ...any) {
	fmt.Fprintf(g.stderr, "%s:%d: ", g.Filename, line)
	fmt.Fprintf(g.stderr, format, args...)
	fmt.Fprintln(g.stderr)
	g.errors++
}

// prepare mirrors main's symbol indexing and Rule_sort before construction of
// the automaton. The first rule remains the start rule even after reordering.
func (g *Grammar) prepare() {
	g.ErrorSymbol = g.byName["error"]
	g.symbol("{default}")
	group := func(s *Symbol) int {
		if s.Type == Multiterminal {
			return 3
		}
		if s.Name[0] > 'Z' {
			return 2
		}
		return 1
	}
	sort.SliceStable(g.Symbols, func(i, j int) bool { return group(g.Symbols[i]) < group(g.Symbols[j]) })
	for i, s := range g.Symbols {
		s.Index = i
	}
	g.NSymbol = len(g.Symbols) - 1
	for g.Symbols[g.NSymbol].Type == Multiterminal {
		g.NSymbol--
	}
	g.NTerminal = 1
	for upper(g.Symbols[g.NTerminal].Name[0]) {
		g.NTerminal++
	}
	g.StartRule = g.Rules[0]
	i := 0
	for _, r := range g.Rules {
		if r.Code != nil {
			r.IRule = i
			i++
		} else {
			r.IRule = -1
		}
	}
	g.NRuleWithAction = i
	for _, r := range g.Rules {
		if r.IRule < 0 {
			r.IRule = i
			i++
		}
	}
	sort.Slice(g.Rules, func(i, j int) bool { return g.Rules[i].IRule < g.Rules[j].IRule })
}

// FindRulePrecedences chooses the FIRST precedence-bearing RHS symbol, not
// yacc's rightmost terminal. Ordered multiterminal members are significant.
func (g *Grammar) FindRulePrecedences() {
	for _, r := range g.Rules {
		for _, s := range r.RHS {
			if r.PrecSymbol != nil {
				break
			}
			if s.Type == Multiterminal {
				for _, sub := range s.Subsymbols {
					if sub.Prec >= 0 {
						r.PrecSymbol = sub
						break
					}
				}
			} else if s.Prec >= 0 {
				r.PrecSymbol = s
			}
		}
	}
}

func upper(c byte) bool { return c >= 'A' && c <= 'Z' }
func lower(c byte) bool { return c >= 'a' && c <= 'z' }
func alpha(c byte) bool { return upper(c) || lower(c) }
func digit(c byte) bool { return c >= '0' && c <= '9' }
func alnum(c byte) bool { return alpha(c) || digit(c) }
func space(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}
