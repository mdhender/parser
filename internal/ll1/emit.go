package ll1

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"go/token"
	"strings"
	"text/template"

	"github.com/mdhender/parser/internal/lemon"
)

//go:embed stack.go.tmpl
var stackTemplate string

//go:embed recursive.go.tmpl
var recursiveTemplate string

func text(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}

// Emit generates a standalone, typed predictive-stack parser, not LR tables.
func (a *Analysis) Emit(pkg string) ([]byte, error) {
	return a.emit(pkg, "stack")
}

func (a *Analysis) emit(pkg, strategy string) ([]byte, error) {
	if len(a.Errors) != 0 {
		return nil, fmt.Errorf("cannot emit an invalid LL(1) grammar")
	}
	g := a.Grammar
	name := text(g.Name, "Parse")
	valid := func(s string) bool { return token.IsIdentifier(s) && s != "_" && !strings.HasPrefix(s, "_glime") }
	if !valid(name) || !valid(pkg) {
		return nil, fmt.Errorf("invalid Go parser or package name")
	}
	types := []string{text(g.TokenType, "int")}
	fields := make(map[*lemon.Symbol]int)
	for _, s := range g.Symbols {
		typ := types[0]
		if s.Type == lemon.Nonterminal && s.Index != 0 {
			typ = text(s.Datatype, text(g.DefaultType, types[0]))
		}
		i := 0
		for i < len(types) && types[i] != typ {
			i++
		}
		if i == len(types) {
			types = append(types, typ)
		}
		fields[s] = i
	}
	var values, constants, actions, table, rhs, names, terminals strings.Builder
	for i, typ := range types {
		fmt.Fprintf(&values, "v%d %s\n", i, typ)
	}
	used := map[string]bool{name: true, name + "Token": true}
	for _, s := range g.Symbols {
		fmt.Fprintf(&names, "%q,", s.Name)
		fmt.Fprintf(&terminals, "%t,", s.Type == lemon.Terminal && s.Index != 0)
		if s.Type == lemon.Terminal && s.Index != 0 {
			n := text(g.TokenPrefix, "") + s.Name
			if !valid(n) || used[n] {
				return nil, fmt.Errorf("invalid or duplicate Go token name %q", n)
			}
			used[n] = true
			fmt.Fprintf(&constants, "%s = %d\n", n, s.Index)
		}
		if s.Index == 0 || s.Type == lemon.Terminal {
			table.WriteString("nil,\n")
			continue
		}
		table.WriteString("{")
		for _, look := range g.Symbols {
			r := a.Table[s][look]
			index := -1
			if r != nil {
				index = r.Index
			}
			fmt.Fprintf(&table, "%d,", index)
		}
		table.WriteString("},\n")
	}
	for _, r := range g.Rules {
		rhs.WriteString("{")
		for _, s := range r.RHS {
			fmt.Fprintf(&rhs, "%d,", s.Index)
		}
		rhs.WriteString("},\n")
		fmt.Fprintf(&actions, "case %d: {\n", r.Index)
		aliases := map[string]bool{}
		bind := func(alias *string, typ, value string) error {
			if alias == nil {
				return nil
			}
			if !valid(*alias) || aliases[*alias] {
				return fmt.Errorf("%s:%d: invalid or duplicate Go alias %q", g.Filename, r.RuleLine, *alias)
			}
			aliases[*alias] = true
			if value == "" {
				fmt.Fprintf(&actions, "var %s %s\n", *alias, typ)
			} else {
				fmt.Fprintf(&actions, "%s := %s\n", *alias, value)
			}
			fmt.Fprintf(&actions, "_ = %s\n", *alias)
			return nil
		}
		if err := bind(r.LHSAlias, types[fields[r.LHS]], ""); err != nil {
			return nil, err
		}
		for i, alias := range r.RHSAlias {
			if err := bind(alias, "", fmt.Sprintf("_glimeFrame.values[%d].v%d", i, fields[r.RHS[i]])); err != nil {
				return nil, err
			}
		}
		fmt.Fprintln(&actions, text(r.Code, ""))
		if r.LHSAlias != nil {
			fmt.Fprintf(&actions, "_glimeResult.v%d = %s\n", fields[r.LHS], *r.LHSAlias)
		}
		actions.WriteString("}\n")
	}
	data := map[string]any{"Package": pkg, "Name": name, "TokenType": types[0], "StartType": types[fields[a.Start]], "StartField": fields[a.Start], "Start": a.Start.Index,
		"Values": values.String(), "Constants": constants.String(), "Actions": actions.String(), "Table": table.String(), "RHS": rhs.String(), "Names": names.String(),
		"Include": text(g.Include, ""), "Code": text(g.ExtraCode, ""), "Accept": text(g.ParseAccept, ""), "SyntaxError": text(g.SyntaxError, ""), "Failure": text(g.ParseFailure, "")}
	skeleton := stackTemplate
	if strategy == "recursive" {
		skeleton = recursiveTemplate
		var functions strings.Builder
		prefix := name + "yy"
		for _, s := range g.Symbols {
			if s.Index == 0 || s.Type == lemon.Terminal {
				continue
			}
			fmt.Fprintf(&functions, "func (_glimeParser *%sparser) parse%d() (%svalue, error) {\n", prefix, s.Index, prefix)
			fmt.Fprintf(&functions, "var _glimeZero %svalue\n_glimeKind, _glimeToken, _glimeErr := _glimeParser.look()\nif _glimeErr != nil { return _glimeZero, _glimeErr }\nswitch _glimeKind {\n", prefix)
			for _, r := range g.Rules {
				if r.LHS != s {
					continue
				}
				var predictions []string
				for _, look := range g.Symbols {
					if a.Table[s][look] == r {
						predictions = append(predictions, fmt.Sprint(look.Index))
					}
				}
				if len(predictions) == 0 {
					continue
				}
				fmt.Fprintf(&functions, "case %s:\n_glimeFrame := &%sframe{rule: %d, values: make([]%svalue, %d)}\n", strings.Join(predictions, ","), prefix, r.Index, prefix, len(r.RHS))
				for i, child := range r.RHS {
					if child.Type == lemon.Terminal {
						fmt.Fprintf(&functions, "_glimeFrame.values[%d], _glimeErr = _glimeParser.match(%d)\n", i, child.Index)
					} else {
						fmt.Fprintf(&functions, "_glimeFrame.values[%d], _glimeErr = _glimeParser.parse%d()\n", i, child.Index)
					}
					functions.WriteString("if _glimeErr != nil { return _glimeZero, _glimeErr }\n")
				}
				fmt.Fprintf(&functions, "return %sreduce(_glimeFrame), nil\n", prefix)
			}
			var expected []string
			for _, look := range g.Symbols {
				if a.Table[s][look] != nil {
					expected = append(expected, look.Name)
				}
			}
			fmt.Fprintf(&functions, "}\nreturn _glimeZero, %sfail(_glimeKind, _glimeToken, %q)\n}\n", prefix, strings.Join(expected, ", "))
		}
		data["Functions"] = functions.String()
		data["Terminals"] = terminals.String()
	}
	// Namespace the skeleton before inserting user code: literals and aliases
	// are never textually rewritten.
	t, err := template.New(strategy).Option("missingkey=error").Parse(strings.ReplaceAll(skeleton, "yy", name+"yy"))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return nil, err
	}
	b, err := format.Source(out.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%s: generated Go syntax: %w", g.Filename, err)
	}
	// Expanding compact action statements moves trailing comments onto new
	// lines. Reparse that layout so gofmt aligns adjacent comment groups too.
	return format.Source(b)
}
