package lemon

import (
	"sort"
	"strconv"
	"strings"
)

// Like SetNew/SetAdd/SetUnion, sets are byte arrays and mutations report change.
// C's SetSize(nterminal+1) allocates nterminal+2 bytes.
type symbolSet []byte

func (s symbolSet) add(i int) bool {
	changed := s[i] == 0
	s[i] = 1
	return changed
}

func (s symbolSet) union(other symbolSet) bool {
	changed := false
	for i, b := range other {
		if b != 0 && s[i] == 0 {
			s[i] = 1
			changed = true
		}
	}
	return changed
}

func (g *Grammar) newSet() symbolSet { return make(symbolSet, g.NTerminal+2) }

// FindFirstSets preserves the separate nullable and FIRST fixed points.
func (g *Grammar) FindFirstSets() {
	for i, s := range g.Symbols[:g.NSymbol] {
		s.Nullable = false
		if i >= g.NTerminal {
			s.First = g.newSet()
		}
	}
	for progress := true; progress; {
		progress = false
		for _, r := range g.Rules {
			if r.LHS.Nullable {
				continue
			}
			nullable := true
			for _, s := range r.RHS {
				if !s.Nullable {
					nullable = false
					break
				}
			}
			if nullable {
				r.LHS.Nullable = true
				progress = true
			}
		}
	}
	for progress := true; progress; {
		progress = false
		for _, r := range g.Rules {
			for _, s := range r.RHS {
				if s.Type == Terminal {
					progress = r.LHS.First.add(s.Index) || progress
					break
				}
				if s.Type == Multiterminal {
					for _, sub := range s.Subsymbols {
						progress = r.LHS.First.add(sub.Index) || progress
					}
					break
				}
				if r.LHS != s {
					progress = r.LHS.First.union(s.First) || progress
				}
				if !s.Nullable {
					break
				}
			}
		}
	}
}

type Config struct {
	Rule              *Rule
	Dot               int
	Follow            symbolSet
	Forward, Backward []*Config
	State             *State
	complete          bool
}

type State struct {
	Number         int
	Basis, Configs []*Config
	Actions        []*Action
}

type configKey struct{ rule, dot int }

// configList replaces C's process-static builder. Lists append in C traversal
// order; maps only deduplicate. Each recursive getstate gets its own builder.
type configList struct {
	g              *Grammar
	basis, configs []*Config
	byKey          map[configKey]*Config
}

func (g *Grammar) configList() *configList {
	return &configList{g: g, byKey: make(map[configKey]*Config)}
}

func (l *configList) add(r *Rule, dot int, basis bool) *Config {
	key := configKey{r.Index, dot}
	c := l.byKey[key]
	if c == nil {
		c = &Config{Rule: r, Dot: dot, Follow: l.g.newSet()}
		l.byKey[key] = c
		l.configs = append(l.configs, c)
		if basis {
			l.basis = append(l.basis, c)
		}
	}
	return c
}

func (l *configList) closure() {
	// Closure appends items while scanning, just like C's linked-list loop.
	for i := 0; i < len(l.configs); i++ {
		c := l.configs[i]
		r := c.Rule
		if c.Dot >= len(r.RHS) {
			continue
		}
		s := r.RHS[c.Dot]
		if s.Type != Nonterminal {
			continue
		}
		if s.Rule == nil && s != l.g.ErrorSymbol {
			l.g.error(r.Line, "Nonterminal \"%s\" has no rules.", s.Name)
		}
		for nr := s.Rule; nr != nil; nr = nr.NextLHS {
			nc := l.add(nr, 0, false)
			j := c.Dot + 1
			for ; j < len(r.RHS); j++ {
				x := r.RHS[j]
				if x.Type == Terminal {
					nc.Follow.add(x.Index)
					break
				}
				if x.Type == Multiterminal {
					for _, sub := range x.Subsymbols {
						nc.Follow.add(sub.Index)
					}
					break
				}
				nc.Follow.union(x.First)
				if !x.Nullable {
					break
				}
			}
			if j == len(r.RHS) {
				c.Forward = append(c.Forward, nc)
			}
		}
	}
}

func sortConfigs(cs []*Config) {
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Rule.Index != cs[j].Rule.Index {
			return cs[i].Rule.Index < cs[j].Rule.Index
		}
		return cs[i].Dot < cs[j].Dot
	})
}

func sameSymbol(a, b *Symbol) bool {
	if a == b {
		return true
	}
	if a.Type != Multiterminal || b.Type != Multiterminal || len(a.Subsymbols) != len(b.Subsymbols) {
		return false
	}
	for i, s := range a.Subsymbols {
		if s != b.Subsymbols[i] {
			return false
		}
	}
	return true
}

func (g *Grammar) startSymbol() *Symbol {
	if g.StartSymbol != nil {
		if s := g.byName[*g.StartSymbol]; s != nil {
			return s
		}
	}
	return g.StartRule.LHS
}

// FindStates constructs the LR(0) graph recursively in discovery order, with
// basis identity independent of lookaheads. Existing bases receive backlinks.
func (g *Grammar) FindStates() {
	s := g.startSymbol()
	if g.StartSymbol != nil && g.byName[*g.StartSymbol] == nil {
		g.error(0, "The specified start symbol \"%s\" is not in a nonterminal of the grammar.  \"%s\" will be used as the start symbol instead.", *g.StartSymbol, s.Name)
	}
	for _, r := range g.Rules {
		for _, rhs := range r.RHS {
			if rhs == s {
				g.error(0, "The start symbol \"%s\" occurs on the right-hand side of a rule. This will result in a parser which does not work properly.", s.Name)
			}
		}
	}
	l := g.configList()
	for r := s.Rule; r != nil; r = r.NextLHS {
		r.LHSStart = true
		l.add(r, 0, true).Follow.add(0)
	}
	g.getstate(l, make(map[string]*State))
}

func (g *Grammar) getstate(l *configList, states map[string]*State) *State {
	sortConfigs(l.basis)
	var key strings.Builder
	for _, c := range l.basis {
		key.WriteString(strconv.Itoa(c.Rule.Index))
		key.WriteByte(':')
		key.WriteString(strconv.Itoa(c.Dot))
		key.WriteByte(';')
	}
	if s := states[key.String()]; s != nil {
		for i, c := range l.basis {
			s.Basis[i].Backward = append(s.Basis[i].Backward, c.Backward...)
		}
		return s
	}
	l.closure()
	sortConfigs(l.configs)
	s := &State{Number: len(g.States), Basis: l.basis, Configs: l.configs}
	g.States = append(g.States, s)
	states[key.String()] = s
	g.buildshifts(s, states)
	return s
}

func (g *Grammar) buildshifts(s *State, states map[string]*State) {
	for _, c := range s.Configs {
		c.complete = false
	}
	for i, c := range s.Configs {
		if c.complete || c.Dot >= len(c.Rule.RHS) {
			continue
		}
		l := g.configList()
		sym := c.Rule.RHS[c.Dot]
		for _, b := range s.Configs[i:] {
			if b.complete || b.Dot >= len(b.Rule.RHS) || !sameSymbol(b.Rule.RHS[b.Dot], sym) {
				continue
			}
			b.complete = true
			nc := l.add(b.Rule, b.Dot+1, true)
			nc.Backward = append(nc.Backward, b)
		}
		next := g.getstate(l, states)
		if sym.Type == Multiterminal {
			for _, sub := range sym.Subsymbols {
				s.addAction(Shift, sub, next, nil)
			}
		} else {
			s.addAction(Shift, sym, next, nil)
		}
	}
}

// FindLinks reverses shift backlinks into forward propagation edges.
// Slice order differs from C's prepended link lists, but the same fixed point
// is reached; no link traversal order is used for state or action numbering.
func (g *Grammar) FindLinks() {
	for _, s := range g.States {
		for _, c := range s.Configs {
			c.State = s
		}
	}
	for _, s := range g.States {
		for _, c := range s.Configs {
			for _, other := range c.Backward {
				other.Forward = append(other.Forward, c)
			}
		}
	}
}

func (g *Grammar) FindFollowSets() {
	for _, s := range g.States {
		for _, c := range s.Configs {
			c.complete = false
		}
	}
	for progress := true; progress; {
		progress = false
		for _, s := range g.States {
			for _, c := range s.Configs {
				if c.complete {
					continue
				}
				for _, other := range c.Forward {
					if other.Follow.union(c.Follow) {
						other.complete = false
						progress = true
					}
				}
				c.complete = true
			}
		}
	}
}
