package ll1_test

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mdhender/parser/internal/lemon"
	"github.com/mdhender/parser/internal/ll1"
)

func parse(t *testing.T, source string) *lemon.Grammar {
	t.Helper()
	var diagnostics bytes.Buffer
	g, err := lemon.ParseGrammar("analysis.y", []byte(source), nil, &diagnostics)
	if err != nil {
		t.Fatalf("ParseGrammar: %v\n%s", err, &diagnostics)
	}
	return g
}

func symbol(t *testing.T, g *lemon.Grammar, name string) *lemon.Symbol {
	t.Helper()
	for _, s := range g.Symbols {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("missing symbol %q", name)
	return nil
}

func wantSet(t *testing.T, set ll1.Set, names ...string) {
	t.Helper()
	got := make([]string, 0, len(set))
	for s, present := range set {
		if present {
			got = append(got, s.Name)
		}
	}
	sort.Strings(got)
	want := append([]string{}, names...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("set = %v, want %v", got, want)
	}
}

func TestNullableFirstFollowSelectAndTable(t *testing.T) {
	g := parse(t, "start ::= a b C.\na ::= A a.\na ::= .\nb ::= B b.\nb ::= .\n")
	a := ll1.Analyze(g)
	if len(a.Errors) != 0 || len(a.Warnings) != 0 {
		t.Fatalf("errors=%v warnings=%v", a.Errors, a.Warnings)
	}
	start, as, bs := symbol(t, g, "start"), symbol(t, g, "a"), symbol(t, g, "b")
	if a.Grammar != g || a.Start != start || !a.Nullable[as] || !a.Nullable[bs] || a.Nullable[start] {
		t.Fatal("wrong identity, start or nullability")
	}
	wantSet(t, a.First[start], "A", "B", "C")
	wantSet(t, a.First[as], "A")
	wantSet(t, a.First[bs], "B")
	wantSet(t, a.First[symbol(t, g, "C")], "C")
	wantSet(t, a.Follow[start], "$")
	wantSet(t, a.Follow[as], "B", "C")
	wantSet(t, a.Follow[bs], "C")
	wantSet(t, a.Select[g.Rules[0]], "A", "B", "C")
	wantSet(t, a.Select[g.Rules[1]], "A")
	wantSet(t, a.Select[g.Rules[2]], "B", "C")
	wantSet(t, a.Select[g.Rules[3]], "B")
	wantSet(t, a.Select[g.Rules[4]], "C")
	for _, r := range g.Rules {
		for token := range a.Select[r] {
			if a.Table[r.LHS][token] != r {
				t.Errorf("wrong table entry for %s, %s", r.LHS.Name, token.Name)
			}
		}
	}
	if a.Table[as][symbol(t, g, "$")] != nil {
		t.Fatal("unexpected table entry")
	}
}

func TestFixedPointsAndNullableStart(t *testing.T) {
	g := parse(t, "start ::= a.\na ::= b c.\nb ::= .\nc ::= d.\nd ::= .\n")
	a := ll1.Analyze(g)
	if len(a.Errors) != 0 {
		t.Fatal(a.Errors)
	}
	for _, name := range []string{"start", "a", "b", "c", "d"} {
		s := symbol(t, g, name)
		if !a.Nullable[s] {
			t.Errorf("%s should be nullable", name)
		}
		wantSet(t, a.First[s])
		wantSet(t, a.Follow[s], "$")
	}
	for _, r := range g.Rules {
		wantSet(t, a.Select[r], "$")
	}
}

func TestExplicitStartAndUnreachableProduction(t *testing.T) {
	g := parse(t, "%start_symbol actual\nunused ::= U.\nactual ::= item.\nitem ::= I.\nunused ::= V.\n")
	a := ll1.Analyze(g)
	if len(a.Errors) != 0 || a.Start.Name != "actual" {
		t.Fatalf("start=%v errors=%v", a.Start, a.Errors)
	}
	wantSet(t, a.Follow[symbol(t, g, "actual")], "$")
	wantSet(t, a.Follow[symbol(t, g, "unused")])
	if len(a.Warnings) != 2 || !strings.Contains(a.Warnings[0], "analysis.y:2: unreachable production unused ::= U .") || !strings.Contains(a.Warnings[1], "analysis.y:5: unreachable production unused ::= V .") {
		t.Fatalf("warnings=%v", a.Warnings)
	}
	bad := ll1.Analyze(parse(t, "%start_symbol missing\nstart ::= A.\n"))
	if !strings.Contains(strings.Join(bad.Errors, "\n"), "%start_symbol \"missing\" has no productions") {
		t.Fatal(bad.Errors)
	}
}

func TestConflicts(t *testing.T) {
	for _, tc := range []struct {
		name, source, kind, overlap string
		first, second               int
	}{
		{"first-first", "start ::= A B.\nstart ::= A C.\n", "FIRST/FIRST", "{A}", 0, 1},
		{"first-follow", "start ::= a A.\na ::= A.\na ::= .\n", "FIRST/FOLLOW", "{A}", 1, 2},
		{"both-nullable", "start ::= a.\na ::= .\na ::= .\n", "FIRST/FOLLOW", "{$}", 1, 2},
		{"indirect-first", "start ::= a.\nstart ::= b.\na ::= A.\nb ::= A.\n", "FIRST/FIRST", "{A}", 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := parse(t, tc.source)
			a := ll1.Analyze(g)
			if len(a.Errors) != 1 {
				t.Fatalf("errors=%v", a.Errors)
			}
			message := a.Errors[0]
			for _, text := range []string{tc.kind, "overlap " + tc.overlap, "rewrite", fmt.Sprintf("analysis.y:%d:", g.Rules[tc.first].RuleLine), fmt.Sprintf("analysis.y:%d:", g.Rules[tc.second].RuleLine)} {
				if !strings.Contains(message, text) {
					t.Errorf("%q missing %q", message, text)
				}
			}
			for token := range a.Select[g.Rules[tc.first]] {
				if a.Select[g.Rules[tc.second]][token] && a.Table[g.Rules[tc.first].LHS][token] != g.Rules[tc.first] {
					t.Fatal("conflict did not retain source-first production")
				}
			}
		})
	}
	g := parse(t, "start ::= A.\nstart ::= A B.\nstart ::= A C.\n")
	if a := ll1.Analyze(g); len(a.Errors) != 3 {
		t.Fatalf("all conflicting pairs must be reported: %v", a.Errors)
	}
}

func TestLeftRecursionAndProductivity(t *testing.T) {
	for _, source := range []string{
		"start ::= start A.\nstart ::= B.\n",
		"start ::= a.\na ::= start A.\na ::= B.\n",
		"start ::= empty a.\nempty ::= .\na ::= start A.\na ::= B.\n",
		"start ::= a.\na ::= start.\n",
	} {
		a := ll1.Analyze(parse(t, source))
		message := strings.Join(a.Errors, "\n")
		for _, text := range []string{"left recursion for start", "analysis.y:1:", "eliminate left recursion"} {
			if !strings.Contains(message, text) {
				t.Errorf("%s: missing %q in %s", source, text, message)
			}
		}
	}
	for _, source := range []string{
		"start ::= A start.\nstart ::= .\n",
		"start ::= a start.\nstart ::= B.\na ::= A.\n",
		"start ::= a.\na ::= b.\nb ::= A b.\nb ::= .\n",
	} {
		if a := ll1.Analyze(parse(t, source)); len(a.Errors) != 0 {
			t.Errorf("not left recursive: %s: %v", source, a.Errors)
		}
	}
	a := ll1.Analyze(parse(t, "start ::= missing.\nloop ::= loop.\n"))
	message := strings.Join(a.Errors, "\n")
	for _, text := range []string{"analysis.y:1: undefined nonterminal missing", "nonproductive nonterminal start", "nonproductive nonterminal loop"} {
		if !strings.Contains(message, text) {
			t.Errorf("missing %q in %s", text, message)
		}
	}
}

func TestUnsupportedLLFeatures(t *testing.T) {
	for _, tc := range []struct{ source, diagnostic string }{
		{"%fallback ID WORD.\nstart ::= ID.\n", "%fallback is unsupported"},
		{"%wildcard ANY.\nstart ::= ANY.\n", "%wildcard is unsupported"},
		{"start ::= error.\n", "error recovery symbol error is unsupported"},
		{"%left PLUS.\nstart ::= PLUS.\n", "precedence does not resolve LL(1) conflicts"},
		{"start ::= A. [PLUS]\n", "precedence override"},
		{"start ::= A|B.\n", "multiterminal"},
		{"%token_class words A B.\nstart ::= words.\n", "multiterminal"},
		{"start ::= A. {NEVER-REDUCE}\n", "NEVER-REDUCE"},
	} {
		t.Run(tc.diagnostic, func(t *testing.T) {
			a := ll1.Analyze(parse(t, tc.source))
			if !strings.Contains(strings.Join(a.Errors, "\n"), tc.diagnostic) {
				t.Fatal(a.Errors)
			}
		})
	}
}

func TestEmitterDirectivesNotValidated(t *testing.T) {
	g := parse(t, "%extra_argument {void *ctx}\n%token_destructor {cleanup($$);}\n%syntax_error {report();}\nstart ::= A. {action();}\n")
	if a := ll1.Analyze(g); len(a.Errors) != 0 {
		t.Fatal(a.Errors)
	}
}

func TestExplainDeterministicAndNoMutation(t *testing.T) {
	g := parse(t, "start ::= B. {action();}\nstart ::= A.\nunused ::= C.\n")
	rules := append([]*lemon.Rule(nil), g.Rules...)
	symbols := append([]*lemon.Symbol(nil), g.Symbols...)
	symbolValues := make([]lemon.Symbol, len(symbols))
	for i, s := range symbols {
		symbolValues[i] = *s
	}
	ruleValues := make([]lemon.Rule, len(rules))
	for i, r := range rules {
		ruleValues[i] = *r
	}
	var first bytes.Buffer
	ll1.Analyze(g).Explain(&first)
	for i := 0; i < 20; i++ {
		var out bytes.Buffer
		ll1.Analyze(g).Explain(&out)
		if out.String() != first.String() {
			t.Fatal("nondeterministic explanation")
		}
	}
	text := first.String()
	for _, want := range []string{"Start: start", "FIRST={B, A}", "analysis.y:1: start ::= B . SELECT={B}", "[start, B]", "Warning: analysis.y:3: unreachable"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
	if strings.Index(text, "start ::= B . SELECT") > strings.Index(text, "start ::= A . SELECT") {
		t.Fatal("productions not explained in source order")
	}
	if !reflect.DeepEqual(rules, g.Rules) || !reflect.DeepEqual(symbols, g.Symbols) || len(g.States) != 0 {
		t.Fatal("analysis reordered grammar or constructed LR states")
	}
	for i, s := range symbols {
		if !reflect.DeepEqual(symbolValues[i], *s) {
			t.Fatalf("analysis mutated symbol %s", s.Name)
		}
	}
	for i, r := range rules {
		if !reflect.DeepEqual(ruleValues[i], *r) {
			t.Fatalf("analysis mutated rule %d", i)
		}
	}
}

func TestNilAndEmptyGrammar(t *testing.T) {
	for _, g := range []*lemon.Grammar{nil, {Filename: "empty.y"}} {
		a := ll1.Analyze(g)
		if len(a.Errors) == 0 {
			t.Fatal("expected an error")
		}
		var out bytes.Buffer
		a.Explain(&out)
		if !strings.Contains(out.String(), "Start: (none)") || !strings.Contains(out.String(), "Error:") {
			t.Fatal(out.String())
		}
	}
}
