package lemon

import "strings"

type parseState int

const (
	waitRule parseState = iota
	waitKeyword
	waitArg
	waitPrecedence
	waitArrow
	inRHS
	lhsAlias1
	lhsAlias2
	lhsAlias3
	rhsAlias1
	rhsAlias2
	precMark1
	precMark2
	resync
	waitDestructor
	waitDatatype
	waitFallback
	waitWildcard
	waitClass
	waitClassToken
	waitToken
)

type parser struct {
	g                    *Grammar
	state                parseState
	fallback, class, lhs *Symbol
	lhsAlias             *string
	rhs                  []*Symbol
	aliases              []*string
	previous             *Rule
	keyword              string
	slot                 **string
	lineSlot             *int
	insertLine           bool
	assoc                Assoc
	precedence           int
}

func (p *parser) token(x string, line int) {
	g := p.g
	c := x[0]
	switch p.state {
	case waitRule:
		switch {
		case c == '%':
			p.state = waitKeyword
		case lower(c):
			p.lhs = g.symbol(x)
			p.rhs = nil
			p.aliases = nil
			p.lhsAlias = nil
			p.state = waitArrow
		case c == '{':
			if p.previous == nil {
				g.error(line, "There is no prior rule upon which to attach the code fragment which begins on this line.")
			} else if p.previous.Code != nil {
				g.error(line, "Code fragment beginning on this line is not the first to follow the previous rule.")
			} else if x == "{NEVER-REDUCE" {
				p.previous.NeverReduce = true
			} else {
				p.previous.Line = line
				code := x[1:]
				p.previous.Code = &code
				p.previous.NoCode = false
			}
		case c == '[':
			p.state = precMark1
		default:
			g.error(line, "Token \"%s\" should be either \"%%\" or a nonterminal name.", x)
		}
	case precMark1:
		if !upper(c) {
			g.error(line, "The precedence symbol must be a terminal.")
		} else if p.previous == nil {
			g.error(line, "There is no prior rule to assign precedence \"[%s]\".", x)
		} else if p.previous.PrecSymbol != nil {
			g.error(line, "Precedence mark on this line is not the first to follow the previous rule.")
		} else {
			p.previous.PrecSymbol = g.symbol(x)
		}
		p.state = precMark2
	case precMark2:
		if c != ']' {
			g.error(line, "Missing \"]\" on precedence mark.")
		}
		p.state = waitRule
	case waitArrow:
		if x == "::=" {
			p.state = inRHS
		} else if c == '(' {
			p.state = lhsAlias1
		} else {
			g.error(line, "Expected to see a \":\" following the LHS symbol \"%s\".", p.lhs.Name)
			p.state = resync
		}
	case lhsAlias1:
		if alpha(c) {
			p.lhsAlias = &x
			p.state = lhsAlias2
		} else {
			g.error(line, "\"%s\" is not a valid alias for the LHS \"%s\"\n", x, p.lhs.Name)
			p.state = resync
		}
	case lhsAlias2:
		if c == ')' {
			p.state = lhsAlias3
		} else {
			g.error(line, "Missing \")\" following LHS alias name \"%s\".", *p.lhsAlias)
			p.state = resync
		}
	case lhsAlias3:
		if x == "::=" {
			p.state = inRHS
		} else {
			g.error(line, "Missing \"->\" following: \"%s(%s)\".", p.lhs.Name, *p.lhsAlias)
			p.state = resync
		}
	case inRHS:
		switch {
		case c == '.':
			r := &Rule{LHS: p.lhs, LHSAlias: p.lhsAlias, RuleLine: line, RHS: p.rhs, RHSAlias: p.aliases, NoCode: true, Index: len(g.Rules), NextLHS: p.lhs.Rule}
			for i, s := range r.RHS {
				if r.RHSAlias[i] != nil {
					s.Content = true
				}
			}
			p.lhs.Rule = r
			g.Rules = append(g.Rules, r)
			p.previous = r
			p.state = waitRule
		case alpha(c):
			if len(p.rhs) >= 1000 {
				g.error(line, "Too many symbols on RHS of rule beginning at \"%s\".", x)
				p.state = resync
			} else {
				p.rhs = append(p.rhs, g.symbol(x))
				p.aliases = append(p.aliases, nil)
			}
		case (c == '|' || c == '/') && len(p.rhs) > 0 && len(x) > 1 && upper(x[1]):
			m := p.rhs[len(p.rhs)-1]
			if m.Type != Multiterminal {
				m = &Symbol{Name: m.Name, Type: Multiterminal, Subsymbols: []*Symbol{m}}
				p.rhs[len(p.rhs)-1] = m
			}
			m.Subsymbols = append(m.Subsymbols, g.symbol(x[1:]))
			if lower(m.Subsymbols[0].Name[0]) {
				g.error(line, "Cannot form a compound containing a non-terminal")
			}
		case c == '(' && len(p.rhs) > 0:
			p.state = rhsAlias1
		default:
			g.error(line, "Illegal character on RHS of rule: \"%s\".", x)
			p.state = resync
		}
	case rhsAlias1:
		if alpha(c) {
			p.aliases[len(p.rhs)-1] = &x
			p.state = rhsAlias2
		} else {
			g.error(line, "\"%s\" is not a valid alias for the RHS symbol \"%s\"\n", x, p.rhs[len(p.rhs)-1].Name)
			p.state = resync
		}
	case rhsAlias2:
		if c == ')' {
			p.state = inRHS
		} else {
			alias := "(null)"
			if p.lhsAlias != nil {
				alias = *p.lhsAlias
			}
			g.error(line, "Missing \")\" following LHS alias name \"%s\".", alias)
			p.state = resync
		}
	case waitKeyword:
		if !alpha(c) {
			g.error(line, "Illegal declaration keyword: \"%s\".", x)
			p.state = resync
			break
		}
		p.keyword = x
		if g.Declarations != nil {
			g.Declarations[x] = line
		}
		p.slot = nil
		p.lineSlot = nil
		p.insertLine = true
		p.state = waitArg
		switch x {
		case "name":
			p.slot = &g.Name
			p.insertLine = false
		case "include":
			p.slot = &g.Include
		case "code":
			p.slot = &g.ExtraCode
		case "token_destructor":
			p.slot = &g.TokenDestructor
		case "default_destructor":
			p.slot = &g.DefaultDestructor
		case "token_prefix":
			p.slot = &g.TokenPrefix
			p.insertLine = false
		case "syntax_error":
			p.slot = &g.SyntaxError
		case "parse_accept":
			p.slot = &g.ParseAccept
		case "parse_failure":
			p.slot = &g.ParseFailure
		case "stack_overflow":
			p.slot = &g.StackOverflow
		case "extra_argument":
			p.slot = &g.ExtraArgument
			p.insertLine = false
		case "extra_context":
			p.slot = &g.ExtraContext
			p.insertLine = false
		case "token_type":
			p.slot = &g.TokenType
			p.insertLine = false
		case "default_type":
			p.slot = &g.DefaultType
			p.insertLine = false
		case "realloc":
			p.slot = &g.Realloc
			p.insertLine = false
		case "free":
			p.slot = &g.Free
			p.insertLine = false
		case "stack_size":
			p.slot = &g.StackSize
			p.insertLine = false
		case "start_symbol":
			p.slot = &g.StartSymbol
			p.insertLine = false
		case "left", "right", "nonassoc":
			p.precedence++
			p.assoc = Left
			if x == "right" {
				p.assoc = Right
			}
			if x == "nonassoc" {
				p.assoc = None
			}
			p.state = waitPrecedence
		case "destructor":
			p.state = waitDestructor
		case "type":
			p.state = waitDatatype
		case "fallback":
			p.fallback = nil
			p.state = waitFallback
		case "token":
			p.state = waitToken
		case "wildcard":
			p.state = waitWildcard
		case "token_class":
			p.state = waitClass
		default:
			g.error(line, "Unknown declaration keyword: \"%%%s\".", x)
			p.state = resync
		}
	case waitDestructor:
		if !alpha(c) {
			g.error(line, "Symbol name missing after %%destructor keyword")
			p.state = resync
		} else {
			s := g.symbol(x)
			p.slot = &s.Destructor
			p.lineSlot = &s.DestLine
			p.insertLine = true
			p.state = waitArg
		}
	case waitDatatype:
		if !alpha(c) {
			g.error(line, "Symbol name missing after %%type keyword")
			p.state = resync
		} else if s := g.byName[x]; s != nil && s.Datatype != nil {
			g.error(line, "Symbol %%type \"%s\" already defined", x)
			p.state = resync
		} else {
			s := g.byName[x]
			if s == nil {
				s = g.symbol(x)
			}
			p.slot = &s.Datatype
			p.insertLine = false
			p.state = waitArg
		}
	case waitPrecedence:
		if c == '.' {
			p.state = waitRule
		} else if !upper(c) {
			g.error(line, "Can't assign a precedence to \"%s\".", x)
		} else {
			s := g.symbol(x)
			if s.Prec >= 0 {
				g.error(line, "Symbol \"%s\" has already be given a precedence.", x)
			} else {
				s.Prec = p.precedence
				s.Assoc = p.assoc
			}
		}
	case waitArg:
		if c != '{' && c != '"' && !alnum(c) {
			g.error(line, "Illegal argument to %%%s: %s", p.keyword, x)
			p.state = resync
			break
		}
		text := x
		if c == '{' || c == '"' {
			text = x[1:]
		}
		old := ""
		if *p.slot != nil {
			old = **p.slot
		}
		if !g.noLineNumbers && p.insertLine && line > 1 && (p.lineSlot == nil || *p.lineSlot != 0) {
			if old != "" && !strings.HasSuffix(old, "\n") {
				old += "\n"
			}
			old += lineDirective(line, g.Filename)
		}
		if p.lineSlot != nil && *p.lineSlot == 0 {
			*p.lineSlot = line
		}
		text = old + text
		*p.slot = &text
		p.state = waitRule
	case waitFallback, waitToken, waitWildcard:
		if c == '.' {
			p.state = waitRule
			break
		}
		if !upper(c) {
			keyword := "fallback"
			if p.state == waitToken {
				keyword = "token"
			}
			if p.state == waitWildcard {
				keyword = "wildcard"
			}
			g.error(line, "%%%s argument \"%s\" should be a token", keyword, x)
			break
		}
		s := g.symbol(x)
		if p.state == waitFallback {
			if p.fallback == nil {
				p.fallback = s
			} else if s.Fallback != nil {
				g.error(line, "More than one fallback assigned to token %s", x)
			} else {
				s.Fallback = p.fallback
				g.HasFallback = true
			}
		} else if p.state == waitWildcard {
			if g.Wildcard == nil {
				g.Wildcard = s
			} else {
				g.error(line, "Extra wildcard to token: %s", x)
			}
		}
	case waitClass:
		if !lower(c) {
			g.error(line, "%%token_class must be followed by an identifier: %s", x)
			p.state = resync
		} else if g.byName[x] != nil {
			g.error(line, "Symbol \"%s\" already used", x)
			p.state = resync
		} else {
			p.class = g.symbol(x)
			p.class.Type = Multiterminal
			p.state = waitClassToken
		}
	case waitClassToken:
		if c == '.' {
			p.state = waitRule
		} else if upper(c) || ((c == '|' || c == '/') && len(x) > 1 && upper(x[1])) {
			if !upper(c) {
				x = x[1:]
			}
			p.class.Subsymbols = append(p.class.Subsymbols, g.symbol(x))
		} else {
			g.error(line, "%%token_class argument \"%s\" should be a token", x)
			p.state = resync
		}
	case resync:
		if c == '.' {
			p.state = waitRule
		}
		if c == '%' {
			p.state = waitKeyword
		}
	}
}
