package lemon

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

//go:embed lempar.go.tmpl
var goTemplate string

// RunGo is the shared entry point for glcc and glemon. It deliberately does not
// route through Run: the C command's flags and emission contract stay intact.
func RunGo(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.SetOutput(stderr)
	pkg := f.String("package", "parser", "generated Go package")
	dir := f.String("d", "", "output directory (must exist)")
	tmpl := f.String("T", "", "Go runtime template override")
	quiet := f.Bool("q", false, "suppress state report")
	uncompressed := f.Bool("c", false, "disable table compression")
	var defines []string
	f.Func("D", "define a grammar preprocessor name", func(s string) error {
		defines = append(defines, strings.SplitN(s, "=", 2)[0])
		return nil
	})
	if err := f.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}
	if f.NArg() != 1 {
		fmt.Fprintln(stderr, "Exactly one filename argument is required.")
		return 1
	}
	if !token.IsIdentifier(*pkg) || *pkg == "_" {
		fmt.Fprintln(stderr, "invalid Go package name")
		return 1
	}
	sort.Strings(defines)
	g := &Grammar{Filename: f.Arg(0), byName: make(map[string]*Symbol), defines: defines, defineUsed: make([]bool, len(defines)), noLineNumbers: true, goTarget: true, stderr: stderr}
	g.symbol("$")
	input, err := os.ReadFile(g.Filename)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	z, ok := g.preprocess(input)
	if !ok {
		return 1
	}
	g.scan(z)
	if g.errors != 0 {
		return 1
	}
	if len(g.Rules) == 0 {
		fmt.Fprintln(stderr, "Empty grammar.")
		return 1
	}
	if g.Realloc != nil || g.Free != nil {
		g.error(0, "%%realloc and %%free are C-only directives")
		return 1
	}
	g.prepare()
	g.FindRulePrecedences()
	g.FindFirstSets()
	g.FindStates()
	g.FindLinks()
	g.FindFollowSets()
	g.FindActions()
	// Go reductions have typed zero-value semantics, not C's implicit union
	// copy. Keep every reduction, including actionless single-symbol rules.
	for _, r := range g.Rules {
		r.NoCode = false
	}
	if !*uncompressed {
		g.CompressTables()
	}
	g.ResortStates()
	if g.NConflict != 0 {
		fmt.Fprintf(stderr, "%d parsing conflicts.\n", g.NConflict)
		return 1
	}
	if g.errors != 0 {
		return 1
	}
	t := g.BuildTables()
	source := goTemplate
	if *tmpl != "" {
		b, e := os.ReadFile(*tmpl)
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
		source = string(b)
	}
	b, err := g.emitGo(*pkg, source, t)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	o := options{flags: map[byte]bool{}}
	if *dir != "" {
		o.outputDir = dir
	}
	if err = os.WriteFile(g.fileName(".go", o), b, 0644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !*quiet {
		g.reportOutput(o)
	}
	if g.errors != 0 {
		return 1
	}
	return 0
}

// A semantic value is a struct of typed fields, not an interface or reflection
// union. Symbols with identical declared types share a field.
func (g *Grammar) emitGo(pkg, source string, t *Tables) ([]byte, error) {
	name := textOr(g.Name, "Parse")
	if !token.IsIdentifier(name) || name == "_" {
		return nil, fmt.Errorf("invalid Go parser name %q", name)
	}
	types := []string{}
	field := make(map[*Symbol]int)
	for _, s := range g.Symbols[:g.NSymbol] {
		typ := textOr(s.Datatype, textOr(g.DefaultType, textOr(g.TokenType, "int")))
		if s.Index < g.NTerminal {
			typ = textOr(g.TokenType, "int")
		}
		if s == g.ErrorSymbol {
			typ = "int"
		}
		if _, err := goparser.ParseExpr(typ); err != nil {
			return nil, fmt.Errorf("type for %s: %w", s.Name, err)
		}
		i := 0
		for i < len(types) && types[i] != typ {
			i++
		}
		if i == len(types) {
			types = append(types, typ)
		}
		field[s] = i
	}
	var values, constants, reductions, destructors strings.Builder
	for i, typ := range types {
		fmt.Fprintf(&values, "v%d %s\n", i, typ)
	}
	for _, s := range g.Symbols[1:g.NTerminal] {
		fmt.Fprintf(&constants, "%s%s = %d\n", textOr(g.TokenPrefix, ""), s.Name, s.Index)
	}
	// Parse declarations with Go's parser so map/function/pointer types work.
	argDecl, argName, err := goParameter(g.ExtraArgument)
	if err != nil {
		return nil, err
	}
	ctxDecl, ctxName, err := goParameter(g.ExtraContext)
	if err != nil {
		return nil, err
	}
	if argName != "" && argName == ctxName {
		return nil, fmt.Errorf("extra argument and context must have distinct names")
	}
	bindings := ""
	for i, n := range []string{argName, ctxName} {
		if n != "" {
			bindings += fmt.Sprintf("%s := _lemonP.extra%d; _ = %s\n", n, i, n)
		}
	}
	lhs, nrhs := []int{}, []int{}
	for _, r := range g.Rules {
		lhs = append(lhs, r.LHS.Index)
		nrhs = append(nrhs, len(r.RHS))
		fmt.Fprintf(&reductions, "case %d: {\n", r.IRule)
		seen := map[string]bool{}
		if r.LHSAlias != nil {
			a := *r.LHSAlias
			seen[a] = true
			fmt.Fprintf(&reductions, "var %s %s; _ = %s\n", a, types[field[r.LHS]], a)
		}
		for i, a := range r.RHSAlias {
			if a == nil {
				continue
			}
			if seen[*a] {
				return nil, fmt.Errorf("rule %d: duplicate alias %s", r.RuleLine, *a)
			}
			seen[*a] = true
			s := r.RHS[i]
			if s.Type == Multiterminal {
				s = s.Subsymbols[0]
			}
			fmt.Fprintf(&reductions, "%s := _lemonP.stack[_lemonBase+%d].value.v%d; _ = %s\n", *a, i, field[s], *a)
		}
		fmt.Fprintln(&reductions, textOr(r.Code, ""))
		if r.LHSAlias != nil {
			fmt.Fprintf(&reductions, "_lemonResult.v%d = %s\n", field[r.LHS], *r.LHSAlias)
		}
		for i, a := range r.RHSAlias {
			if a == nil {
				fmt.Fprintf(&reductions, "_lemonP.destroy(_lemonP.stack[_lemonBase+%d])\n", i)
			}
		}
		fmt.Fprintln(&reductions, "}")
	}
	for _, s := range g.Symbols[1:g.NSymbol] {
		d := s.Destructor
		if d == nil {
			if s.Type == Terminal {
				d = g.TokenDestructor
			} else {
				d = g.DefaultDestructor
			}
		}
		if d != nil {
			fmt.Fprintf(&destructors, "case %d: { _lemonValue := _lemonEntry.value.v%d; _ = _lemonValue\n%s\n}\n", s.Index, field[s], goDestructor(*d))
		}
	}
	fallback := make([]int, g.NTerminal)
	for _, s := range g.Symbols[:g.NTerminal] {
		visited := map[*Symbol]bool{s: true}
		for x := s.Fallback; x != nil; x = x.Fallback {
			if visited[x] {
				return nil, fmt.Errorf("fallback cycle at %s", s.Name)
			}
			visited[x] = true
		}
		if s.Fallback != nil {
			fallback[s.Index] = s.Fallback.Index
		}
	}
	wildcard, errorSymbol := -1, -1
	if g.Wildcard != nil {
		wildcard = g.Wildcard.Index
	}
	if g.ErrorSymbol != nil {
		errorSymbol = g.ErrorSymbol.Index
	}
	depth, err := strconv.Atoi(strings.TrimSpace(textOr(g.StackSize, "0")))
	if err != nil || depth < 0 {
		return nil, fmt.Errorf("%%stack_size requires a nonnegative integer")
	}
	data := map[string]any{
		"Package": pkg, "Name": name, "Include": textOr(g.Include, ""), "Code": textOr(g.ExtraCode, ""),
		"Values": values.String(), "Constants": constants.String(), "Reductions": reductions.String(), "Destructors": destructors.String(),
		"TokenType": textOr(g.TokenType, "int"), "TokenField": field[g.Symbols[0]],
		"ArgDecl": argDecl, "ArgName": argName, "CtxDecl": ctxDecl, "CtxName": ctxName, "Bindings": bindings,
		"ArgType": strings.TrimSpace(strings.TrimPrefix(argDecl, argName)), "CtxType": strings.TrimSpace(strings.TrimPrefix(ctxDecl, ctxName)),
		"Action": t.Action, "Lookahead": t.Lookahead, "Shift": t.Shift, "Reduce": t.Reduce, "Default": t.Default,
		"Fallback": fallback, "LHS": lhs, "NRHS": nrhs, "Wildcard": wildcard, "ErrorSymbol": errorSymbol,
		"NTerminal": g.NTerminal, "NState": len(g.States), "MinReduce": g.MinReduce, "ErrAction": g.ErrAction, "AccAction": g.AccAction,
		"Depth": depth, "SyntaxError": textOr(g.SyntaxError, ""), "Accept": textOr(g.ParseAccept, ""), "Failure": textOr(g.ParseFailure, ""), "Overflow": textOr(g.StackOverflow, ""),
	}
	// Namespace only the runtime skeleton, never user code or action literals.
	source = strings.ReplaceAll(source, "yy", name+"yy")
	tm, err := template.New("Go parser").Funcs(template.FuncMap{"ints": func(v []int) string {
		var b strings.Builder
		for _, n := range v {
			fmt.Fprintf(&b, "%d,", n)
		}
		return b.String()
	}}).Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err = tm.Execute(&b, data); err != nil {
		return nil, err
	}
	out, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("generated Go syntax: %w", err)
	}
	return out, nil
}

func goParameter(decl *string) (string, string, error) {
	if decl == nil {
		return "", "", nil
	}
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, "parameter.go", "package p; func f("+*decl+") {}", 0)
	if err != nil {
		return "", "", err
	}
	params := f.Decls[0].(*ast.FuncDecl).Type.Params.List
	if len(params) != 1 || len(params[0].Names) != 1 || params[0].Names[0].Name == "_" {
		return "", "", fmt.Errorf("extra argument/context must declare one named Go parameter")
	}
	name := params[0].Names[0].Name
	if strings.HasPrefix(name, "_lemon") || name == "TOKEN" {
		return "", "", fmt.Errorf("extra parameter name %q is reserved", name)
	}
	if _, variadic := params[0].Type.(*ast.Ellipsis); variadic {
		return "", "", fmt.Errorf("extra parameter cannot be variadic")
	}
	var typ bytes.Buffer
	if err := format.Node(&typ, fset, params[0].Type); err != nil {
		return "", "", err
	}
	return name + " " + typ.String(), name, nil
}

// Replace $$ only outside Go comments and literals. Go aliases need no textual
// replacement at all: reduction cases bind ordinary typed local variables.
func goDestructor(code string) string {
	var out strings.Builder
	fset := token.NewFileSet()
	file := fset.AddFile("destructor", -1, len(code))
	var scan scanner.Scanner
	// '$' is intentionally not a Go token. Ignore scanner diagnostics here;
	// format.Source validates the final Go after $$ has been substituted.
	scan.Init(file, []byte(code), func(token.Position, string) {}, scanner.ScanComments)
	last := 0
	for {
		pos, tok, lit := scan.Scan()
		if tok == token.EOF {
			break
		}
		i := file.Offset(pos)
		if tok == token.ILLEGAL && lit == "$" && strings.HasPrefix(code[i:], "$$") {
			out.WriteString(code[last:i])
			out.WriteString("_lemonValue")
			last = i + 2
			scan.Scan() // The adjacent second '$'.
		}
	}
	out.WriteString(code[last:])
	return out.String()
}
