package lemon

import (
	"fmt"
	"io"
	"strings"
)

// translateCode mirrors translate_code: labels are expanded even inside C
// comments and strings. The returned flag requests the yylhsminor temporary.
// Call stackUnion first so that every symbol's Dtnum has been assigned.
func (g *Grammar) translateCode(r *Rule) bool {
	nrhs := len(r.RHS)
	used := make([]bool, nrhs)
	lhsused := false
	dontUseRhs0 := false
	zSkip := -1
	zOvwrt := ""
	hasDestructor := func(s *Symbol) bool {
		if s.Type == Terminal {
			return g.TokenDestructor != nil
		}
		return g.DefaultDestructor != nil || s.Destructor != nil
	}
	if r.Code == nil {
		code := "\n"
		r.Code = &code
		r.Line = r.RuleLine
		r.NoCode = true
	} else {
		r.NoCode = false
	}

	lhsdirect := true
	switch {
	case nrhs == 0:
	case r.RHSAlias[0] == nil:
		if hasDestructor(r.RHS[0]) {
			prefix := fmt.Sprintf("  yy_destructor(yypParser,%d,&yymsp[%d].minor);\n", r.RHS[0].Index, 1-nrhs)
			r.CodePrefix = &prefix
			r.NoCode = false
		}
	case r.LHSAlias == nil:
	case *r.LHSAlias == *r.RHSAlias[0]:
		lhsused = true
		used[0] = true
		if r.LHS.Dtnum != r.RHS[0].Dtnum {
			g.error(r.RuleLine, "%s(%s) and %s(%s) share the same label but have different datatypes.", r.LHS.Name, *r.LHSAlias, r.RHS[0].Name, *r.RHSAlias[0])
		}
	default:
		zOvwrt = fmt.Sprintf("/*%s-overwrites-%s*/", *r.LHSAlias, *r.RHSAlias[0])
		zSkip = strings.Index(*r.Code, zOvwrt)
		lhsdirect = zSkip >= 0
	}
	zLhs := fmt.Sprintf("yymsp[%d].minor.yy%d", 1-nrhs, r.LHS.Dtnum)
	if !lhsdirect {
		zLhs = fmt.Sprintf("yylhsminor.yy%d", r.LHS.Dtnum)
	}

	code := *r.Code
	var expanded []byte
	for cp := 0; cp < len(code); cp++ {
		if cp == zSkip {
			expanded = append(expanded, zOvwrt...)
			cp += len(zOvwrt) - 1
			dontUseRhs0 = true
			continue
		}
		if alpha(code[cp]) && (cp == 0 || (!alnum(code[cp-1]) && code[cp-1] != '_')) {
			xp := cp + 1
			for xp < len(code) && (alnum(code[xp]) || code[xp] == '_') {
				xp++
			}
			label := code[cp:xp]
			if r.LHSAlias != nil && label == *r.LHSAlias {
				expanded = append(expanded, zLhs...)
				cp = xp
				lhsused = true
			} else {
				for i, alias := range r.RHSAlias {
					if alias == nil || label != *alias {
						continue
					}
					if i == 0 && dontUseRhs0 {
						g.error(r.RuleLine, "Label %s used after '%s'.", *r.RHSAlias[0], zOvwrt)
					} else if cp != 0 && code[cp-1] == '@' {
						expanded = expanded[:len(expanded)-1]
						expanded = append(expanded, fmt.Sprintf("yymsp[%d].major", i-nrhs+1)...)
					} else {
						s := r.RHS[i]
						if s.Type == Multiterminal {
							s = s.Subsymbols[0]
						}
						expanded = append(expanded, fmt.Sprintf("yymsp[%d].minor.yy%d", i-nrhs+1, s.Dtnum)...)
					}
					cp = xp
					used[i] = true
					break
				}
			}
		}
		// The C loop copies the delimiter after an expanded identifier too.
		if cp < len(code) {
			expanded = append(expanded, code[cp])
		}
	}
	if len(expanded) != 0 {
		translated := string(expanded)
		r.Code = &translated
	}

	if r.LHSAlias != nil && !lhsused {
		g.error(r.RuleLine, "Label \"%s\" for \"%s(%s)\" is never used.", *r.LHSAlias, r.LHS.Name, *r.LHSAlias)
	}
	var suffix strings.Builder
	for i, s := range r.RHS {
		alias := r.RHSAlias[i]
		if alias != nil {
			if i > 0 {
				if r.LHSAlias != nil && *r.LHSAlias == *alias {
					g.error(r.RuleLine, "%s(%s) has the same label as the LHS but is not the left-most symbol on the RHS.", s.Name, *alias)
				}
				for j := 0; j < i; j++ {
					if r.RHSAlias[j] != nil && *r.RHSAlias[j] == *alias {
						g.error(r.RuleLine, "Label %s used for multiple symbols on the RHS of a rule.", *alias)
						break
					}
				}
			}
			if !used[i] {
				g.error(r.RuleLine, "Label %s for \"%s(%s)\" is never used.", *alias, s.Name, *alias)
			}
		} else if i > 0 && hasDestructor(s) {
			fmt.Fprintf(&suffix, "  yy_destructor(yypParser,%d,&yymsp[%d].minor);\n", s.Index, i-nrhs+1)
		}
	}
	if !lhsdirect {
		fmt.Fprintf(&suffix, "  yymsp[%d].minor.yy%d = %s;\n", 1-nrhs, r.LHS.Dtnum, zLhs)
	}
	if suffix.Len() != 0 {
		text := suffix.String()
		r.CodeSuffix = &text
		r.NoCode = false
	}
	return !lhsdirect
}

// stackUnion mirrors print_stack_union, including its datatype hash placement
// and resulting semantic-value field numbers. Output line counting is left to
// the caller's writer, rather than keeping a separate counter here.
func (g *Grammar) stackUnion(w io.Writer, mh bool) {
	arraysize := g.NSymbol * 2
	types := make([]*string, arraysize)
	for _, s := range g.Symbols[:g.NSymbol] {
		if s == g.ErrorSymbol {
			s.Dtnum = arraysize + 1
			continue
		}
		if s.Type != Nonterminal || (s.Datatype == nil && g.DefaultType == nil) {
			s.Dtnum = 0
			continue
		}
		datatype := s.Datatype
		if datatype == nil {
			datatype = g.DefaultType
		}
		// C's ISSPACE is byte-oriented; Unicode whitespace is not trimmed.
		stddt := strings.TrimFunc(*datatype, func(r rune) bool { return r < 128 && space(byte(r)) })
		if g.TokenType != nil && stddt == *g.TokenType {
			s.Dtnum = 0
			continue
		}
		var hash uint32
		for j := 0; j < len(stddt); j++ {
			// The pinned C baseline uses unsigned hash arithmetic and signed
			// plain char on its recorded platform.
			hash = hash*53 + uint32(int32(int8(stddt[j])))
		}
		hash = (hash & 0x7fffffff) % uint32(arraysize)
		for types[hash] != nil && *types[hash] != stddt {
			hash++
			if hash >= uint32(arraysize) {
				hash = 0
			}
		}
		s.Dtnum = int(hash) + 1
		if types[hash] == nil {
			types[hash] = &stddt
		}
	}
	name := "Parse"
	if g.Name != nil {
		name = *g.Name
	}
	tokentype := "void*"
	if g.TokenType != nil {
		tokentype = *g.TokenType
	}
	if mh {
		fmt.Fprintln(w, "#if INTERFACE")
	}
	fmt.Fprintf(w, "#define %sTOKENTYPE %s\n", name, tokentype)
	if mh {
		fmt.Fprintln(w, "#endif")
	}
	fmt.Fprintf(w, "typedef union {\n  int yyinit;\n  %sTOKENTYPE yy0;\n", name)
	for i, datatype := range types {
		if datatype != nil {
			fmt.Fprintf(w, "  %s yy%d;\n", *datatype, i+1)
		}
	}
	if g.ErrorSymbol != nil && g.ErrorSymbol.UseCount != 0 {
		fmt.Fprintf(w, "  int yy%d;\n", g.ErrorSymbol.Dtnum)
	}
	fmt.Fprintln(w, "} YYMINORTYPE;")
}
