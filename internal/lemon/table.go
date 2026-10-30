package lemon

import (
	"fmt"
	"io"
	"sort"
)

const noOffset = -2147483647

// CompressTables follows the three C passes: default reductions, incoming
// SHIFTREDUCE actions, then bypassing no-code single-symbol nonterminal rules.
func (g *Grammar) CompressTables() {
	for _, s := range g.States {
		var best *Rule
		nbest := 0
		wildcard := false
		for i, a := range s.Actions {
			if a.Type == Shift && a.Symbol == g.Wildcard {
				wildcard = true
			}
			if a.Type != Reduce || a.Rule.LHSStart || a.Rule == best {
				continue
			}
			n := 1
			for _, b := range s.Actions[i+1:] {
				if b.Type == Reduce && b.Rule != best && b.Rule == a.Rule {
					n++
				}
			}
			if n > nbest {
				nbest, best = n, a.Rule
			}
		}
		if nbest < 1 || wildcard {
			continue
		}
		found := false
		for _, a := range s.Actions {
			if a.Type == Reduce && a.Rule == best {
				if !found {
					a.Symbol = g.symbol("{default}")
					found = true
				} else {
					a.Type = NotUsed
				}
			}
		}
		s.sortActions()
		auto := true
		for _, a := range s.Actions {
			if a.Type == Shift || (a.Type == Reduce && a.Rule != best) {
				auto = false
				break
			}
		}
		if auto {
			s.AutoReduce, s.DefaultReduce = true, best
		}
	}
	for _, s := range g.States {
		for _, a := range s.Actions {
			if a.Type == Shift && a.State.AutoReduce && a.State.DefaultReduce != nil {
				a.Type, a.Rule = ShiftReduce, a.State.DefaultReduce
				a.State = nil
			}
		}
	}
	for _, s := range g.States {
		for _, a := range s.Actions {
			// C revisits this same action after each substitution, so a
			// chain may collapse through more than one nonterminal.
			for a.Type == ShiftReduce && a.Rule.NoCode && len(a.Rule.RHS) == 1 && a.Symbol.Index >= g.NTerminal {
				var next *Action
				for _, b := range s.Actions {
					if b != a && b.Symbol == a.Rule.LHS {
						next = b
						break
					}
				}
				// Like C's assert(ap2!=0), the LHS goto must exist here.
				a.Opt = next.Symbol
				a.Type, a.State, a.Rule = next.Type, next.State, next.Rule
			}
		}
	}
}

func (g *Grammar) computeAction(a *Action) int {
	switch a.Type {
	case Shift:
		return a.State.Number
	case ShiftReduce:
		if a.Symbol.Index >= g.NTerminal && (g.ErrorSymbol == nil || a.Symbol.Index != g.ErrorSymbol.Index) {
			return g.MinReduce + a.Rule.IRule
		}
		return g.MinShiftReduce + a.Rule.IRule
	case Reduce:
		return g.MinReduce + a.Rule.IRule
	case Error:
		return g.ErrAction
	case Accept:
		return g.AccAction
	default:
		return -1
	}
}

// ResortStates runs before numeric action boundaries are assigned. Default
// reductions therefore store generated rule numbers, not final action codes.
func (g *Grammar) ResortStates() {
	for _, s := range g.States {
		s.NTknAct, s.NNtAct = 0, 0
		s.IDfltReduce = -1
		s.ITknOfst, s.INtOfst = noOffset, noOffset
		for _, a := range s.Actions {
			code := g.computeAction(a)
			if code < 0 {
				continue
			}
			switch {
			case a.Symbol.Index < g.NTerminal:
				s.NTknAct++
			case a.Symbol.Index < g.NSymbol:
				s.NNtAct++
			default:
				s.IDfltReduce = code
			}
		}
	}
	tail := g.States[1:]
	sort.Slice(tail, func(i, j int) bool {
		a, b := tail[i], tail[j]
		if a.NNtAct != b.NNtAct {
			return a.NNtAct > b.NNtAct
		}
		if a.NTknAct != b.NTknAct {
			return a.NTknAct > b.NTknAct
		}
		return a.Number > b.Number
	})
	for i, s := range g.States {
		s.Number = i
	}
	g.NXState = len(g.States)
	for g.NXState > 1 && g.States[g.NXState-1].AutoReduce {
		g.NXState--
	}
}

type lookaheadAction struct{ lookahead, action int }

// acttab keeps the original allocation growth: the hole-search bound uses
// allocated space, whereas duplicate and accidental-match scans use nAction.
type acttab struct {
	actions, transaction []lookaheadAction
	nAction              int
	min, max, minAction  int
	nsymbol, nterminal   int
}

func (p *acttab) add(lookahead, action int) {
	if len(p.transaction) == 0 {
		p.min, p.max, p.minAction = lookahead, lookahead, action
	} else {
		if lookahead > p.max {
			p.max = lookahead
		}
		if lookahead < p.min {
			p.min, p.minAction = lookahead, action
		}
	}
	p.transaction = append(p.transaction, lookaheadAction{lookahead, action})
}

func (p *acttab) insert(safe bool) int {
	n := p.nsymbol + 1
	if p.nAction+n >= len(p.actions) {
		old := len(p.actions)
		p.actions = append(p.actions, make([]lookaheadAction, p.nAction+n+20)...)
		for i := old; i < len(p.actions); i++ {
			p.actions[i] = lookaheadAction{-1, -1}
		}
	}
	end := 0
	if safe {
		end = p.min
	}
	i := p.nAction - 1
	for ; i >= end; i-- {
		if p.actions[i].lookahead != p.min || p.actions[i].action != p.minAction {
			continue
		}
		matches := true
		for _, a := range p.transaction {
			k := a.lookahead - p.min + i
			if k < 0 || k >= p.nAction || p.actions[k] != a {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		n = 0
		for j, a := range p.actions[:p.nAction] {
			if a.lookahead >= 0 && a.lookahead == j+p.min-i {
				n++
			}
		}
		if n == len(p.transaction) {
			break
		}
	}
	if i < end {
		for i = end; i < len(p.actions)-p.max; i++ {
			if p.actions[i].lookahead >= 0 {
				continue
			}
			fits := true
			for _, a := range p.transaction {
				k := a.lookahead - p.min + i
				if k < 0 || p.actions[k].lookahead >= 0 {
					fits = false
					break
				}
			}
			if !fits {
				continue
			}
			for j, a := range p.actions[:p.nAction] {
				if a.lookahead == j+p.min-i {
					fits = false
					break
				}
			}
			if fits {
				break
			}
		}
	}
	for _, a := range p.transaction {
		k := a.lookahead - p.min + i
		p.actions[k] = a
		if k >= p.nAction {
			p.nAction = k + 1
		}
	}
	if safe && i+p.nterminal >= p.nAction {
		p.nAction = i + p.nterminal + 1
	}
	p.transaction = p.transaction[:0]
	return i - p.min
}

func (p *acttab) actionSize() int {
	n := p.nAction
	for n > 0 && p.actions[n-1].lookahead < 0 {
		n--
	}
	return n
}

// Tables contains the numeric arrays prepared by ReportTable, without C text
// rendering. Raw packing and offset bounds are retained for differential checks.
type Tables struct {
	Action, Lookahead, Shift, Reduce, Default    []int
	MinTknOfst, MaxTknOfst, MinNtOfst, MaxNtOfst int
	packed                                       *acttab
}

func (g *Grammar) BuildTables() *Tables {
	g.MinShiftReduce = len(g.States)
	g.ErrAction = g.MinShiftReduce + len(g.Rules)
	g.AccAction = g.ErrAction + 1
	g.NoAction = g.AccAction + 1
	g.MinReduce = g.NoAction + 1
	g.MaxAction = g.MinReduce + len(g.Rules)
	t := &Tables{packed: &acttab{nsymbol: g.NSymbol, nterminal: g.NTerminal}}
	type axset struct {
		state        *State
		token        bool
		count, order int
	}
	ax := make([]axset, 0, g.NXState*2)
	for _, s := range g.States[:g.NXState] {
		ax = append(ax, axset{s, true, s.NTknAct, len(ax)}, axset{s, false, s.NNtAct, len(ax) + 1})
	}
	sort.Slice(ax, func(i, j int) bool {
		if ax[i].count != ax[j].count {
			return ax[i].count > ax[j].count
		}
		return ax[i].order < ax[j].order
	})
	for _, x := range ax {
		if x.count <= 0 {
			break
		}
		for _, a := range x.state.Actions {
			if x.token {
				if a.Symbol.Index >= g.NTerminal {
					continue
				}
			} else if a.Symbol.Index < g.NTerminal || a.Symbol.Index == g.NSymbol {
				continue
			}
			if code := g.computeAction(a); code >= 0 {
				t.packed.add(a.Symbol.Index, code)
			}
		}
		offset := t.packed.insert(x.token)
		if x.token {
			x.state.ITknOfst = offset
			t.MinTknOfst = min(t.MinTknOfst, offset)
			t.MaxTknOfst = max(t.MaxTknOfst, offset)
		} else {
			x.state.INtOfst = offset
			t.MinNtOfst = min(t.MinNtOfst, offset)
			t.MaxNtOfst = max(t.MaxNtOfst, offset)
		}
	}
	for _, r := range g.Rules {
		r.DoesReduce = false
	}
	for _, s := range g.States[:g.NXState] {
		for _, a := range s.Actions {
			if a.Type == Reduce || a.Type == ShiftReduce {
				a.Rule.DoesReduce = true
			}
		}
	}
	for _, a := range t.packed.actions[:t.packed.actionSize()] {
		code := a.action
		if code < 0 {
			code = g.NoAction
		}
		t.Action = append(t.Action, code)
	}
	for _, a := range t.packed.actions[:t.packed.nAction] {
		lookahead := a.lookahead
		if lookahead < 0 {
			lookahead = g.NSymbol
		}
		t.Lookahead = append(t.Lookahead, lookahead)
	}
	for len(t.Lookahead) < g.NTerminal+len(t.Action) {
		t.Lookahead = append(t.Lookahead, g.NTerminal)
	}
	n := g.NXState
	for n > 0 && g.States[n-1].ITknOfst == noOffset {
		n--
	}
	for _, s := range g.States[:n] {
		offset := s.ITknOfst
		if offset == noOffset {
			offset = len(t.Action)
		}
		t.Shift = append(t.Shift, offset)
	}
	n = g.NXState
	for n > 0 && g.States[n-1].INtOfst == noOffset {
		n--
	}
	for _, s := range g.States[:n] {
		offset := s.INtOfst
		if offset == noOffset {
			offset = t.MinNtOfst - 1
		}
		t.Reduce = append(t.Reduce, offset)
	}
	for _, s := range g.States[:g.NXState] {
		code := g.ErrAction
		if s.IDfltReduce >= 0 {
			code = s.IDfltReduce + g.MinReduce
		}
		t.Default = append(t.Default, code)
	}
	return t
}

func (g *Grammar) TableSnapshot(w io.Writer, t *Tables) {
	fmt.Fprintf(w, "tables\t%d\t%d\t%d\t%d\t%d\t%d\n", g.NSymbol, g.NTerminal, len(g.Rules), len(g.States), g.NXState, g.NConflict)
	fmt.Fprintf(w, "codes\t%d\t%d\t%d\t%d\t%d\t%d\n", g.MinShiftReduce, g.ErrAction, g.AccAction, g.NoAction, g.MinReduce, g.MaxAction)
	fmt.Fprintf(w, "bounds\t%d\t%d\t%d\t%d\n", t.MinTknOfst, t.MaxTknOfst, t.MinNtOfst, t.MaxNtOfst)
	fmt.Fprintf(w, "sizes\t%d\t%d\n", t.packed.actionSize(), t.packed.nAction)
	for _, r := range g.Rules {
		fmt.Fprintf(w, "rule\t%d\t%d\n", r.Index, boolInt(r.DoesReduce))
	}
	for _, s := range g.States {
		def := -1
		if s.DefaultReduce != nil {
			def = s.DefaultReduce.Index
		}
		fmt.Fprintf(w, "state\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", s.Number, s.NTknAct, s.NNtAct, s.ITknOfst, s.INtOfst, s.IDfltReduce, boolInt(s.AutoReduce), def)
		for _, a := range s.Actions {
			var target any = "-"
			switch a.Type {
			case Shift, SHResolved, SSConflict:
				target = a.State.Number
			case Reduce, SRConflict, RRConflict, RDResolved, NotUsed, ShiftReduce:
				target = a.Rule.Index
			}
			opt := -1
			if a.Opt != nil {
				opt = a.Opt.Index
			}
			fmt.Fprintf(w, "action\t%d\t%d\t%v\t%d\t%d\n", a.Symbol.Index, a.Type, target, opt, g.computeAction(a))
		}
	}
	for i, a := range t.packed.actions[:t.packed.nAction] {
		fmt.Fprintf(w, "entry\t%d\t%d\t%d\n", i, a.lookahead, a.action)
	}
	for _, array := range []struct {
		name   string
		values []int
	}{
		{"yy-action", t.Action}, {"yy-lookahead", t.Lookahead}, {"yy-shift", t.Shift}, {"yy-reduce", t.Reduce}, {"yy-default", t.Default},
	} {
		for i, value := range array.values {
			fmt.Fprintf(w, "%s\t%d\t%d\n", array.name, i, value)
		}
	}
}
