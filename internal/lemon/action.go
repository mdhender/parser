package lemon

import "sort"

// ActionType values match enum e_action in the pinned source. Resolved and
// conflicting actions remain in the list; they are not live parser actions.
type ActionType int

const (
	Shift ActionType = iota
	Accept
	Reduce
	Error
	SSConflict
	SRConflict
	RRConflict
	SHResolved
	RDResolved
)

type Action struct {
	Symbol *Symbol
	Type   ActionType
	State  *State
	Rule   *Rule
	order  int
}

func (s *State) addAction(kind ActionType, sym *Symbol, next *State, r *Rule) {
	s.Actions = append(s.Actions, &Action{Symbol: sym, Type: kind, State: next, Rule: r, order: len(s.Actions)})
}

func (g *Grammar) FindActions() {
	for _, s := range g.States {
		for _, c := range s.Configs {
			if c.Dot == len(c.Rule.RHS) {
				for j := 0; j < g.NTerminal; j++ {
					if c.Follow[j] != 0 {
						s.addAction(Reduce, g.Symbols[j], nil, c.Rule)
					}
				}
			}
		}
	}
	g.States[0].addAction(Accept, g.startSymbol(), nil, nil)
	for _, s := range g.States {
		sort.Slice(s.Actions, func(i, j int) bool {
			a, b := s.Actions[i], s.Actions[j]
			if a.Symbol.Index != b.Symbol.Index {
				return a.Symbol.Index < b.Symbol.Index
			}
			if a.Type != b.Type {
				return a.Type < b.Type
			}
			if a.Type == Reduce && a.Rule.Index != b.Rule.Index {
				return a.Rule.Index < b.Rule.Index
			}
			// C actioncmp breaks ties by reverse allocation address. Use
			// reverse creation order without depending on Go heap addresses.
			return a.order > b.order
		})
		for i, a := range s.Actions {
			for _, b := range s.Actions[i+1:] {
				if b.Symbol != a.Symbol {
					break
				}
				g.NConflict += resolveConflict(a, b)
			}
		}
	}
	for _, r := range g.Rules {
		r.CanReduce = false
	}
	for _, s := range g.States {
		for _, a := range s.Actions {
			if a.Type == Reduce {
				a.Rule.CanReduce = true
			}
		}
	}
	for _, r := range g.Rules {
		if !r.CanReduce {
			g.error(r.RuleLine, "This rule can not be reduced.\n")
		}
	}
}

// resolveConflict follows C's pairwise, mutating resolution. In particular,
// equal nonassoc changes only the shift to ERROR and leaves the reduce intact.
func resolveConflict(a, b *Action) int {
	if a.Type == Shift && b.Type == Shift {
		b.Type = SSConflict
		return 1
	}
	if a.Type == Shift && b.Type == Reduce {
		x, y := a.Symbol, b.Rule.PrecSymbol
		if y == nil || x.Prec < 0 || y.Prec < 0 {
			b.Type = SRConflict
			return 1
		}
		switch {
		case x.Prec > y.Prec:
			b.Type = RDResolved
		case x.Prec < y.Prec:
			a.Type = SHResolved
		case x.Assoc == Right:
			b.Type = RDResolved
		case x.Assoc == Left:
			a.Type = SHResolved
		default:
			a.Type = Error
		}
	} else if a.Type == Reduce && b.Type == Reduce {
		x, y := a.Rule.PrecSymbol, b.Rule.PrecSymbol
		if x == nil || y == nil || x.Prec < 0 || y.Prec < 0 || x.Prec == y.Prec {
			b.Type = RRConflict
			return 1
		}
		if x.Prec > y.Prec {
			b.Type = RDResolved
		} else {
			a.Type = RDResolved
		}
	}
	return 0
}
