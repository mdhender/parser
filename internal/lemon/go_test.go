package lemon_test

import (
	"bytes"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdhender/parser/internal/lemon"
)

// Compile and execute the public generated API in a module with no dependency
// on the generator. All expected results are derived from the token streams.
func TestGoParsers(t *testing.T) {
	cases := []struct{ name, grammar, test string }{
		{"calculator", `%name Calc
%token_prefix TK_
%token_type {int}
%extra_argument {out *int}
%left PLUS MINUS.
%left TIMES.
start ::= expr(A). { *out = A }
expr(A) ::= expr(B) PLUS expr(C). { A = B+C }
expr(A) ::= expr(B) MINUS expr(C). { A = B-C }
expr(A) ::= expr(B) TIMES expr(C). { A = B*C }
expr(A) ::= LP expr(B) RP. { A = B }
expr(A) ::= NUM(B). { A = B; _ = "A B }"; _ = ` + "`A B }`" + ` /* A B */ }
`, `func TestRuntime(t *testing.T) {
 for _, tc := range []struct{ tokens, values []int; want int }{
  {[]int{TK_NUM,TK_MINUS,TK_NUM,TK_MINUS,TK_NUM}, []int{19,0,6,0,2},11},
  {[]int{TK_NUM,TK_PLUS,TK_NUM,TK_TIMES,TK_NUM}, []int{2,0,3,0,7},23},
  {[]int{TK_LP,TK_NUM,TK_PLUS,TK_NUM,TK_RP,TK_TIMES,TK_NUM}, []int{0,2,0,3,0,0,7},35},
 } { out := -1; p := NewCalc(); for i,k := range tc.tokens { p.Parse(k,tc.values[i],&out) }; p.Parse(0,0,&out); if out != tc.want { t.Fatalf("got %d want %d",out,tc.want) }; p.Close() }
}`},
		{"types_empty", `%name Typed
%token_type {string}
%default_type {int}
%type words {[]string}
%extra_context {out *[]string}
%include {import "strings"}
%code {const marker = "trailing"}
start ::= words(A). { *out = A }
words(A) ::= . { A = []string{marker} }
words(A) ::= words(B) WORD(C). { A = append(B, strings.ToUpper(C)) }
`, `func TestRuntime(t *testing.T) {
 var out []string; p:=NewTyped(&out); p.Parse(WORD,"first"); p.Parse(WORD,"second"); p.Parse(0,""); if len(out)!=3 || out[0]!="trailing" || out[1]!="FIRST" || out[2]!="SECOND" { t.Fatal(out) }
 p.Parse(0,""); if len(out)!=1 { t.Fatal(out) }
}`},
		{"fallback_wildcard", `%name Tokens
%token_type {int}
%extra_argument {out *[]int}
%fallback ID KEY.
%wildcard ANY.
%token OTHER.
start ::= items.
items ::= .
items ::= items ID(A). { *out=append(*out,A+100) }
items ::= items ANY(A). { *out=append(*out,A+200) }
`, `func TestRuntime(t *testing.T) {
 var out []int; p:=NewTokens(); p.Parse(KEY,3,&out); p.Parse(OTHER,7,&out); p.Parse(0,0,&out); if len(out)!=2 || out[0]!=103 || out[1]!=207 { t.Fatal(out) }
}`},
		{"recovery", `%name Recover
%token_type {int}
%extra_context {out *Result}
%include {type Result struct { Sum, Errors, Accept, Fail, Destroy int }}
%syntax_error {out.Errors++; out.Destroy += TOKEN}
%parse_accept {out.Accept++}
%parse_failure {out.Fail++}
%token_destructor {out.Destroy += $$; _ = "$$"; _ = ` + "`$$ }`" + ` /* $$ */ }
%token BAD.
start ::= items.
items ::= .
items ::= items NUM(A) SEMI. { out.Sum += A }
items ::= items error SEMI.
`, `func TestRuntime(t *testing.T) {
 out:=Result{}; p:=NewRecover(&out)
 for _, pair:=range [][2]int{{NUM,5},{SEMI,0},{NUM,0},{BAD,7},{SEMI,0},{NUM,11},{SEMI,0},{0,0}} {p.Parse(pair[0],pair[1])}
 if out.Sum!=16 || out.Errors!=1 || out.Accept!=1 || out.Fail!=0 || out.Destroy!=14 {t.Fatal(out)}
 p.Parse(NUM,13); p.Close(); if out.Destroy!=27 {t.Fatal(out)}
 p.Parse(NUM,0); p.Parse(0,0); p.Parse(NUM,0); p.Parse(0,0)
 if out.Errors!=2 || out.Fail!=2 {t.Fatal(out)}
}`},
		{"failure", `%extra_argument {out *Result}
%include {type Result struct { Errors, Fail, Accept, Destroy int }}
%syntax_error {out.Errors++}
%parse_failure {out.Fail++}
%parse_accept {out.Accept++}
%token_destructor {out.Destroy += $$}
%token BAD.
start ::= NUM NUM.
`, `func TestRuntime(t *testing.T) {
 out:=Result{}; var p Parse; p.Parse(NUM,5,&out); p.Parse(0,0,&out)
 if out.Fail!=1 || out.Errors!=1 || out.Destroy!=5 {t.Fatal(out)}
 p.Parse(NUM,2,&out); p.Parse(BAD,7,&out); p.Parse(NUM,3,&out); p.Parse(0,0,&out)
 if out.Accept!=1 || out.Destroy!=17 {t.Fatal(out)}
}`},
		{"stack", `%name Stack
%stack_size 4
%extra_context {out *Result}
%include {type Result struct { Overflow, Destroy, Accept int }}
%stack_overflow {out.Overflow++}
%token_destructor {out.Destroy += $$}
%parse_accept {out.Accept++}
start ::= nested.
nested ::= LP nested RP.
nested ::= .
`, `func TestRuntime(t *testing.T) {
 out:=Result{}; p:=NewStack(&out); for i:=0;i<4;i++ {p.Parse(LP,1)}
 if out.Overflow!=1 || out.Destroy!=3 {t.Fatal(out)}
 p.Parse(0,0); if out.Accept!=1 {t.Fatal(out)}
}`},
		{"dynamic_stack", `%name Deep
%extra_context {out *int}
start ::= nested(A). { *out=A }
nested(A) ::= LP nested(B) RP. { A=B+1 }
nested(A) ::= . { A=0 }
`, `func TestRuntime(t *testing.T) {
 out:=0; p:=NewDeep(&out); for i:=0;i<2048;i++ {p.Parse(LP,0)}; for i:=0;i<2048;i++ {p.Parse(RP,0)}; p.Parse(0,0); if out!=2048 {t.Fatal(out)}
}`},
		{"names_and_parameters", `%extra_context {p *int}
%extra_argument {value int}
start ::= expr(result). { *p=result+value }
expr(result) ::= NUM(base). { result=base }
`, `func TestRuntime(t *testing.T) {
 out:=0; parser:=NewParse(&out); parser.Parse(NUM,7,1); parser.Parse(0,0,4); if out!=11 {t.Fatal(out)}
}`},
		{"typed_destructors", `%name Resources
%token_type {int}
%default_type {string}
%type item {*int}
%extra_context {out *int}
%destructor item { *out += *$$ }
%default_destructor { *out += len($$) }
start ::= item END.
item(A) ::= NUM(B). { A=&B }
`, `func TestRuntime(t *testing.T) {
 out:=0; p:=NewResources(&out); p.Parse(NUM,17); p.Parse(END,0); p.Close(); if out!=17 {t.Fatal(out)}
}`},
		{"empty_stack_limit", `%stack_size 1
%extra_context {out *int}
%stack_overflow { *out += 10 }
start ::= . { *out += 1 }
`, `func TestRuntime(t *testing.T) {
 out:=0; p:=NewParse(&out); p.Parse(0,0); if out!=10 {t.Fatal(out)}
}`},
		{"no_actions", "start ::= TOKEN.\n", `func TestRuntime(t *testing.T) { var p Parse; p.Parse(TOKEN,0); p.Parse(0,0); p.Close() }`},
	}
	for _, tc := range cases {
		for _, compress := range []bool{true, false} {
			t.Run(tc.name+map[bool]string{true: "/compressed", false: "/uncompressed"}[compress], func(t *testing.T) {
				dir := t.TempDir()
				grammar := filepath.Join(dir, "grammar.y")
				writeGoTestFile(t, grammar, tc.grammar)
				args := []string{"glcc", "-q", grammar}
				if !compress {
					args = []string{"glcc", "-q", "-c", grammar}
				}
				var stdout, stderr bytes.Buffer
				if code := lemon.RunGo(args, &stdout, &stderr); code != 0 {
					t.Fatalf("generate %d: %s", code, &stderr)
				}
				generated, err := os.ReadFile(filepath.Join(dir, "grammar.go"))
				if err != nil {
					t.Fatal(err)
				}
				formatted, err := format.Source(generated)
				if err != nil || !bytes.Equal(formatted, generated) {
					t.Fatal("not gofmt formatted", err)
				}
				args[0] = "glemon"
				if code := lemon.RunGo(args, &stdout, &stderr); code != 0 {
					t.Fatal(stderr.String())
				}
				again, _ := os.ReadFile(filepath.Join(dir, "grammar.go"))
				if !bytes.Equal(generated, again) {
					t.Fatal("nondeterministic generation")
				}
				writeGoTestFile(t, filepath.Join(dir, "go.mod"), "module generated\n\ngo 1.27.3\n")
				writeGoTestFile(t, filepath.Join(dir, "grammar_test.go"), "package parser\nimport \"testing\"\n"+tc.test)
				cmd := exec.Command("go", "test", "-count=1", "./...")
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "CGO_ENABLED=0")
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("generated runtime: %v\n%s", err, out)
				}
			})
		}
	}
}

func writeGoTestFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGoCommands(t *testing.T) {
	for _, name := range []string{"glcc", "glemon"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			binary := filepath.Join(dir, name)
			cmd := exec.Command("go", "build", "-o", binary, "../../cmd/"+name)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("build: %v %s", err, out)
			}
			grammar := filepath.Join(dir, "tiny.y")
			writeGoTestFile(t, grammar, "start ::= TOKEN.\n")
			cmd = exec.Command(binary, "-package", "example", grammar)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("invoke: %v %s", err, out)
			}
			b, err := os.ReadFile(filepath.Join(dir, "tiny.go"))
			if err != nil || !bytes.Contains(b, []byte("package example")) {
				t.Fatal("missing parser", err)
			}
			if _, err = os.Stat(filepath.Join(dir, "tiny.out")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGoDiagnostics(t *testing.T) {
	for _, tc := range []struct{ grammar, want string }{
		{"start ::= .\n%realloc {malloc}\n", "C-only"},
		{"start ::= .\n%stack_size bogus\n", "nonnegative integer"},
		{"start ::= X.\n%fallback X Y.\n%fallback Y X.\n", "fallback cycle"},
		{"start(A) ::= X(A).\n", "duplicate alias"},
		{"start ::= .\n%name {bad name}\n", "invalid Go parser name"},
		{"start ::= X. { this is not Go }\n", "generated Go syntax"},
		{"start ::= X X.\n%unknown x\n", "Unknown declaration"},
		{"start ::= X.\ntrailing ::= X\n", "Unexpected end of grammar"},
		{"start ::= X.\n%extra_argument {args ...int}\n", "cannot be variadic"},
		{"start ::= X.\n%extra_argument {out int}\n%extra_context {out int}\n", "distinct names"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "bad.y")
			writeGoTestFile(t, path, tc.grammar)
			var out, err bytes.Buffer
			if lemon.RunGo([]string{"glcc", "-q", path}, &out, &err) == 0 || !strings.Contains(err.String(), tc.want) {
				t.Fatalf("got %s want %s", &err, tc.want)
			}
			if _, e := os.Stat(filepath.Join(dir, "bad.go")); !os.IsNotExist(e) {
				t.Fatal("invalid grammar wrote parser")
			}
		})
	}
}

func TestGoOptions(t *testing.T) {
	dir := t.TempDir()
	grammar := filepath.Join(dir, "input.y")
	writeGoTestFile(t, grammar, "%ifdef ENABLED\nstart ::= NUM.\n%else\n%unknown x\n%endif\n")
	// Explicitly overriding with the embedded skeleton must preserve output.
	var out, stderr bytes.Buffer
	args := []string{"glcc", "-q", "-D", "ENABLED", "-d", dir, grammar}
	if lemon.RunGo(args, &out, &stderr) != 0 {
		t.Fatal(stderr.String())
	}
	before, err := os.ReadFile(filepath.Join(dir, "input.go"))
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := filepath.Abs("lempar.go.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	args = append([]string{"glcc", "-T", tmpl}, args[1:]...)
	if lemon.RunGo(args, &out, &stderr) != 0 {
		t.Fatal(stderr.String())
	}
	after, err := os.ReadFile(filepath.Join(dir, "input.go"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("template override changed output", err)
	}
	for _, flags := range [][]string{
		{"-T", filepath.Join(dir, "missing")}, {"-d", filepath.Join(dir, "missing")},
		{"-package", "not-a-package"}, {"-language", "go"},
	} {
		stderr.Reset()
		args := append([]string{"glcc", "-D", "ENABLED"}, flags...)
		args = append(args, grammar)
		if lemon.RunGo(args, &out, &stderr) == 0 || stderr.Len() == 0 {
			t.Fatalf("accepted invalid flags %v", flags)
		}
	}
}

func TestGoTutorial(t *testing.T) {
	doc, err := os.ReadFile("../../docs/tutorials/go-parser.md")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, block := range []struct{ language, file string }{{"lemon", "calc.y"}, {"go", "main.go"}} {
		_, rest, ok := strings.Cut(string(doc), "```"+block.language+"\n")
		if !ok {
			t.Fatalf("missing %s block", block.language)
		}
		body, _, ok := strings.Cut(rest, "\n```")
		if !ok {
			t.Fatal("unclosed tutorial block")
		}
		writeGoTestFile(t, filepath.Join(dir, block.file), body+"\n")
	}
	var stdout, stderr bytes.Buffer
	if lemon.RunGo([]string{"glcc", "-package", "main", filepath.Join(dir, "calc.y")}, &stdout, &stderr) != 0 {
		t.Fatal(stderr.String())
	}
	cmd := exec.Command("go", "run", filepath.Join(dir, "main.go"), filepath.Join(dir, "calc.go"))
	cmd.Env = append(os.Environ(), "GOPROXY=off", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil || string(out) != "Result: 11\n" {
		t.Fatalf("tutorial: %v %s", err, out)
	}
}
