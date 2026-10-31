package ll1_test

import (
	"bytes"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdhender/parser/internal/ll1"
)

// Exercise generated public APIs in dependency-free modules, not generator
// internals. Asymmetric values detect reversed children and reused frames.
func TestGeneratedParsers(t *testing.T) {
	calc, err := os.ReadFile("../../docs/tutorials/ll1-parser/calc.y")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, grammar, test string }{
		{"calculator", string(calc), `func TestRuntime(t *testing.T) {
 for _, tc := range []struct { in []CalcToken; want int }{
  {[]CalcToken{{TK_NUM,19},{TK_MINUS,0},{TK_NUM,6},{TK_MINUS,0},{TK_NUM,2}},11},
  {[]CalcToken{{TK_NUM,2},{TK_PLUS,0},{TK_NUM,3},{TK_TIMES,0},{TK_NUM,7}},23},
  {[]CalcToken{{TK_LP,0},{TK_NUM,2},{TK_PLUS,0},{TK_NUM,3},{TK_RP,0},{TK_TIMES,0},{TK_NUM,7}},35},
 } { got,err:=Calc(tc.in); if err!=nil || got!=tc.want {t.Fatalf("got %d, %v; want %d",got,err,tc.want)} }
 for _, in := range [][]CalcToken{nil, {{TK_LP,0},{TK_NUM,2}}, {{TK_NUM,2},{TK_NUM,3}}, {{TK_PLUS,0}}, {{-1,0}}, {{999,0}}, {{1,0}}, {{0,0}}, {{TK_NUM,2},{0,0}}} {
  if _,err:=Calc(in);err==nil {t.Fatalf("accepted %v",in)}
 }
}`},
		{"completion", `%include {var events []int; var literal, raw string}
%parse_accept {events = append(events, 99)}
%syntax_error {events = append(events, -TOKEN)}
%parse_failure {events = append(events, -99)}
start(A) ::= pair(B) pair(C). { A = B*100+C; events=append(events,A) }
pair(A) ::= LP pair(B) RP. { A=B+1; events=append(events,A) }
pair(A) ::= NUM(B). { A=B; events=append(events,A); literal="A B }"; /* A B } */ raw=` + "`A B }`" + ` // A B }
}
`, `func TestRuntime(t *testing.T) {
 got,err:=Parse([]ParseToken{{LP,0},{NUM,3},{RP,0},{NUM,8}})
 if err!=nil || got!=408 || !reflect.DeepEqual(events,[]int{3,4,8,408,99}) { t.Fatal(got,err,events) }
 if literal!="A B }" || raw!="A B }" { t.Fatal(literal,raw) }
 events=nil; _,err=Parse([]ParseToken{{LP,0},{NUM,5},{NUM,7}})
 if err==nil || !strings.Contains(err.Error(),"expected RP") || !reflect.DeepEqual(events,[]int{5,-7,-99}) { t.Fatal(err,events) }
 events=nil; _,err=Parse([]ParseToken{{NUM,2},{NUM,9},{NUM,4}})
 if err==nil || !strings.Contains(err.Error(),"expected $") || !reflect.DeepEqual(events,[]int{2,9,209,-4,-99}) { t.Fatal(err,events) }
 events=nil; _,err=Parse([]ParseToken{{NUM,5}})
 if err==nil || !strings.Contains(err.Error(),"unexpected $") || !reflect.DeepEqual(events,[]int{5,0,-99}) { t.Fatal(err,events) }
}`},
		{"heterogeneous", `%name Words
%token_type {string}
%default_type {int}
%type words {[]string}
%start_symbol words
%include {import "strings"}
%code {const suffix = "!"}
unused ::= .
words(A) ::= WORD(B) words(C). { A=append([]string{strings.ToUpper(B)+suffix},C...) }
words(A) ::= . { A=[]string{} }
`, `func TestRuntime(t *testing.T) {
 got,err:=Words([]WordsToken{{WORD,"first"},{WORD,"next"}})
 if err!=nil || !reflect.DeepEqual(got,[]string{"FIRST!","NEXT!"}) { t.Fatal(got,err) }
 got,err=Words(nil); if err!=nil || len(got)!=0 { t.Fatal(got,err) }
}`},
		{"deep", `start(A) ::= LP start(B) RP. { A=B+1 }
start(A) ::= . { A=0 }
`, `func TestRuntime(t *testing.T) {
 in:=make([]ParseToken,0,8192);for i:=0;i<4096;i++ {in=append(in,ParseToken{Kind:LP})};for i:=0;i<4096;i++ {in=append(in,ParseToken{Kind:RP})}
 got,err:=Parse(in);if err!=nil || got!=4096 {t.Fatal(got,err)}
 got,err=Parse(nil);if err!=nil || got!=0 {t.Fatal(got,err)}
}`},
		{"actionless", "start ::= NUM tail.\ntail ::= .\n", `func TestRuntime(t *testing.T) {
 got,err:=Parse([]ParseToken{{NUM,13}});if err!=nil || got!=0 {t.Fatal(got,err)}
}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			grammar := filepath.Join(dir, "grammar.y")
			if err := os.WriteFile(grammar, []byte(tc.grammar), 0600); err != nil {
				t.Fatal(err)
			}
			var out, diagnostics bytes.Buffer
			if status := ll1.Run([]string{"glime", "generate", grammar}, &out, &diagnostics); status != 0 {
				t.Fatal(status, diagnostics.String())
			}
			generated, err := os.ReadFile(filepath.Join(dir, "grammar.go"))
			if err != nil {
				t.Fatal(err)
			}
			formatted, err := format.Source(generated)
			if err != nil || !bytes.Equal(generated, formatted) {
				t.Fatalf("not formatted (%v):\n%s\n--- second pass ---\n%s", err, generated, formatted)
			}
			if status := ll1.Run([]string{"glime", "generate", grammar}, &out, &diagnostics); status != 0 {
				t.Fatal(diagnostics.String())
			}
			again, _ := os.ReadFile(filepath.Join(dir, "grammar.go"))
			if !bytes.Equal(generated, again) {
				t.Fatal("nondeterministic generation")
			}
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module generated\n\ngo 1.27.3\n"), 0600); err != nil {
				t.Fatal(err)
			}
			imports := "import \"testing\"\n"
			if strings.Contains(tc.test, "reflect.") {
				imports += "import \"reflect\"\n"
			}
			if strings.Contains(tc.test, "strings.") {
				imports += "import \"strings\"\n"
			}
			if err := os.WriteFile(filepath.Join(dir, "runtime_test.go"), []byte("package parser\n"+imports+tc.test), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "test", "-count=1", "./...")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOPROXY=off", "GOWORK=off", "CGO_ENABLED=0")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	dir := t.TempDir()
	grammar := filepath.Join(dir, "grammar.y")
	if err := os.WriteFile(grammar, []byte("%ifdef ENABLE\nstart ::= NUM.\n%else\nstart ::= .\n%endif\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args       []string
		status     int
		diagnostic string
	}{
		{[]string{"check", grammar}, 0, ""},
		{[]string{"-D", "ENABLE", "explain", grammar}, 0, ""},
		{[]string{"-strategy=recursive", "generate", grammar}, 1, "not implemented"},
		{[]string{"-strategy=unknown", "check", grammar}, 1, "unknown strategy"},
		{[]string{"check", "-strategy=stack", grammar}, 1, "Usage:"},
		{[]string{"-package=for", "check", grammar}, 1, "invalid Go package"},
		{[]string{"-h"}, 0, "Usage:"},
		{nil, 1, "Usage:"},
	} {
		var out, diagnostics bytes.Buffer
		status := ll1.Run(append([]string{"glime"}, tc.args...), &out, &diagnostics)
		if status != tc.status || !strings.Contains(diagnostics.String(), tc.diagnostic) {
			t.Fatalf("%v: %d %s", tc.args, status, &diagnostics)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "grammar.go")); !os.IsNotExist(err) {
		t.Fatal("check/explain wrote output")
	}
	bin := filepath.Join(dir, "glime")
	cmd := exec.Command("go", "build", "-o", bin, "../../cmd/glime")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	cmd = exec.Command(bin, "-package=example", "generate", grammar)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	generated, err := os.ReadFile(filepath.Join(dir, "grammar.go"))
	if err != nil || !bytes.Contains(generated, []byte("package example")) {
		t.Fatal(err, string(generated))
	}
}

func TestUnsupportedAndInvalid(t *testing.T) {
	for _, grammar := range []string{
		"%extra_argument {out *int}\nstart ::= NUM.", "%token_destructor {}\nstart ::= NUM.",
		"%stack_size 20\nstart ::= NUM.", "%left NUM.\nstart ::= NUM.",
		"%fallback NUM.\nstart ::= NUM.", "%wildcard ANY.\nstart ::= NUM.",
		"%token_class many NUM WORD.\nstart ::= NUM.",
		"%token_class many NUM WORD.\nmany ::= NUM.",
		"start(A) ::= NUM(A).", "start(for) ::= NUM.", "%name for\nstart ::= NUM.",
		"start(A) ::= NUM. { A = }", "start ::= NUM", "%token_type {[] }\nstart ::= NUM.",
	} {
		t.Run(grammar, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "bad.y")
			if err := os.WriteFile(path, []byte(grammar), 0600); err != nil {
				t.Fatal(err)
			}
			var out, diagnostics bytes.Buffer
			if ll1.Run([]string{"glime", "generate", path}, &out, &diagnostics) == 0 || diagnostics.Len() == 0 {
				t.Fatal("accepted", grammar)
			}
			if _, err := os.Stat(filepath.Join(dir, "bad.go")); !os.IsNotExist(err) {
				t.Fatal("wrote invalid parser")
			}
		})
	}
}
