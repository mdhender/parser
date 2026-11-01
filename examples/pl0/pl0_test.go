package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/parser/examples/pl0/lalr"
	"github.com/mdhender/parser/examples/pl0/lang"
	"github.com/mdhender/parser/examples/pl0/ll1"
	"github.com/mdhender/parser/examples/pl0/recursive"
	"github.com/mdhender/parser/internal/lemon"
	llgenerator "github.com/mdhender/parser/internal/ll1"
)

var frontends = []struct {
	name  string
	parse func(string) (*lang.Block, error)
}{
	{"glcc", lalr.Parse},
	{"glime", ll1.Parse},
	{"glime-recursive", recursive.Parse},
}

func TestPrograms(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"squares", "", "1\n4\n9\n16\n25\n55\n"},
		{"scopes", "", "11\n3\n3\n"},
		{"factorial", "", "720\n"},
		{"empty", ".", ""},
		{"empty_statements", "begin ; ; if 1 = 1 then ; while 1 # 1 do ; end.", ""},
		{"arithmetic", "begin write 19-6-2; write 96/4/3; write 2+3*7; write -(2+3)*4; write +7; write (-7)/3 end.", "11\n8\n23\n-20\n7\n-2\n"},
		{"relations", `begin
 if 2 = 2 then write 1; if 2 = 3 then write 99;
 if 2 # 3 then write 2; if 2 # 2 then write 99;
 if 2 < 3 then write 3; if 3 < 3 then write 99;
 if 3 <= 3 then write 4; if 4 <= 3 then write 99;
 if 4 > 3 then write 5; if 3 > 3 then write 99;
 if 3 >= 3 then write 6; if 2 >= 3 then write 99;
 if odd (-3) then write 7; if odd 4 then write 99
end.`, "1\n2\n3\n4\n5\n6\n7\n"},
		{"declarations", "const a=7,b=2,c=3; var x,y,z; begin x:=a-b-c; y:=x*10; z:=y+1; write z end.", "21\n"},
		{"fresh_locals", "procedure p; var x; begin x:=x+1; write x end; begin call p; call p end.", "1\n1\n"},
		{"constant_shadow", "const x=7; procedure p; const x=11; write x; begin call p; write x end.", "11\n7\n"},
		{"recursive_capture", `var n;
procedure descend;
var saved;
procedure show; write saved;
begin saved:=n; if n>0 then begin n:=n-1; call descend end; call show end;
begin n:=3; call descend end.`, "0\n1\n2\n3\n"},
		{"mutual_recursion", `var n;
procedure a; begin if n > 0 then begin n:=n-1; call b end end;
procedure b; begin write n; if n > 0 then begin n:=n-1; call a end end;
begin n:=5; call a end.`, "4\n2\n0\n"},
		{"comments", "/* leading\n comment */ begin // line\n write 8 /* between */ / 2 end. // EOF", "4\n"},
		{"int64", "begin write 9223372036854775807+1; write (0-9223372036854775807-1)/(0-1) end.", "-9223372036854775808\n-9223372036854775808\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			if source == "" {
				data, err := os.ReadFile(filepath.Join("programs", tc.name+".pl0"))
				if err != nil {
					t.Fatal(err)
				}
				source = string(data)
			}
			var first *lang.Block
			for _, frontend := range frontends {
				t.Run(frontend.name, func(t *testing.T) {
					block, err := frontend.parse(source)
					if err != nil {
						t.Fatal(err)
					}
					if first == nil {
						first = block
					} else if !reflect.DeepEqual(first, block) {
						t.Fatal("frontends produced different ASTs (including source locations)")
					}
					// The same AST can be executed again: runtime cells never live in it.
					for range 2 {
						var output bytes.Buffer
						if err := lang.Execute(block, &output); err != nil {
							t.Fatal(err)
						}
						if output.String() != tc.want {
							t.Fatalf("output %q; want %q", output.String(), tc.want)
						}
					}
				})
			}
		})
	}
}

// Independent expected shapes catch the possibility that all frontends agree
// on the wrong associativity or construct an evaluator instead of a real AST.
func TestExpressionAST(t *testing.T) {
	var shape func(*lang.Expr) string
	shape = func(e *lang.Expr) string {
		if e.Right == nil {
			return e.Token.Text
		}
		if e.Left == nil {
			return fmt.Sprintf("(%s %s)", e.Token.Text, shape(e.Right))
		}
		return fmt.Sprintf("(%s %s %s)", e.Token.Text, shape(e.Left), shape(e.Right))
	}
	for _, frontend := range frontends {
		t.Run(frontend.name, func(t *testing.T) {
			block, err := frontend.parse("begin write 19-6-2; write 96/4/3; write 2+3*7; write -2*3+4 end.")
			if err != nil {
				t.Fatal(err)
			}
			for i, want := range []string{"(- (- 19 6) 2)", "(/ (/ 96 4) 3)", "(+ 2 (* 3 7))", "(+ (- (* 2 3)) 4)"} {
				if got := shape(block.Statement.Body[i].Expr); got != want {
					t.Errorf("expression %d: %s; want %s", i, got, want)
				}
			}
		})
	}
}

func TestRejectSource(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"", "1:1:"},
		{"begin\nwrite 1", "2:8:"},
		{"begin\n write ; end.", "2:8:"},
		{"begin end. write 1", "1:12:"},
		{"var x,; .", "1:7:"},
		{"const a=-1; .", "1:9:"},
		{"if 1 then .", "1:6:"},
		{"begin end", "1:10:"},
		{"write 2 + .", "1:11:"},
		{"write 2 * -3.", "1:11:"},
		{"write 9223372036854775808.", "integer literal out of int64 range"},
		{"write 12abc.", "identifier cannot start with a digit"},
		{"/* no end", "1:1: unterminated comment"},
		{"write @.", "1:7: unexpected character"},
	} {
		for _, frontend := range frontends {
			t.Run(frontend.name+"/"+tc.source, func(t *testing.T) {
				block, err := frontend.parse(tc.source)
				if err == nil || !strings.Contains(err.Error(), tc.want) || block != nil {
					t.Fatalf("got block %v, error %v; want %q", block, err, tc.want)
				}
			})
		}
	}
}

func TestSemanticAndRuntimeErrors(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"var x,x; .", "duplicate"},
		{"const x=1; var x; .", "duplicate"},
		{"var x; procedure x; ; .", "duplicate"},
		{"begin write 1; x:=2 end.", "undefined"},
		{"procedure unused; write missing; write 1.", "undefined"},
		{"begin write 1; if 1=2 then call missing end.", "undefined"},
		{"const x=1; x:=2.", "variable"},
		{"var x; call x.", "procedure"},
		{"procedure p; ; write p.", "value"},
		{"procedure p; ; p:=1.", "variable"},
		{"write 1/0.", "division by zero"},
	} {
		for _, frontend := range frontends {
			t.Run(frontend.name+"/"+tc.source, func(t *testing.T) {
				block, err := frontend.parse(tc.source)
				if err != nil {
					t.Fatal(err)
				}
				var output bytes.Buffer
				err = lang.Execute(block, &output)
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v; want %q", err, tc.want)
				}
				if output.Len() != 0 {
					t.Fatalf("invalid program produced output: %q", output.String())
				}
			})
		}
	}
	writeErr := errors.New("output unavailable")
	for _, frontend := range frontends {
		block, err := frontend.parse("write 7.")
		if err != nil {
			t.Fatal(err)
		}
		if err := lang.Execute(block, failingWriter{writeErr}); !errors.Is(err, writeErr) {
			t.Fatalf("%s: lost writer error: %v", frontend.name, err)
		}
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestGeneratedParsersCurrent(t *testing.T) {
	for _, frontend := range []struct {
		dir, command, grammar, strategy string
		run                             func([]string, io.Writer, io.Writer) int
	}{
		{"lalr", "glcc", "lalr/grammar.y", "", lemon.RunGo},
		{"ll1", "glime", "ll1/grammar.y", "stack", llgenerator.Run},
		{"recursive", "glime", "ll1/grammar.y", "recursive", llgenerator.Run},
	} {
		t.Run(frontend.dir, func(t *testing.T) {
			dir := t.TempDir()
			args := []string{frontend.command, "-package", frontend.dir, "-d", dir}
			if frontend.command == "glcc" {
				args = append(args, "-q")
			} else {
				args = append(args, "-strategy="+frontend.strategy, "generate")
			}
			args = append(args, frontend.grammar)
			var out, diagnostics bytes.Buffer
			if code := frontend.run(args, &out, &diagnostics); code != 0 {
				t.Fatalf("generate: %d %s", code, &diagnostics)
			}
			want, err := os.ReadFile(filepath.Join(frontend.dir, "grammar.go"))
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "grammar.go"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatal("generated parser is stale; run go generate ./examples/pl0/...")
			}
		})
	}
}

func TestCLI(t *testing.T) {
	for _, frontend := range frontends {
		parser := frontend.name
		var out, diagnostics bytes.Buffer
		if code := run([]string{"-parser", parser, "programs/squares.pl0"}, &out, &diagnostics); code != 0 || out.String() != "1\n4\n9\n16\n25\n55\n" || diagnostics.Len() != 0 {
			t.Fatalf("%s: %d %q %s", parser, code, &out, &diagnostics)
		}
	}
	var out, diagnostics bytes.Buffer
	if code := run([]string{"programs/factorial.pl0"}, &out, &diagnostics); code != 0 || out.String() != "720\n" || diagnostics.Len() != 0 {
		t.Fatalf("default parser: %d %q %s", code, &out, &diagnostics)
	}
	for _, args := range [][]string{nil, {"-parser", "unknown", "programs/squares.pl0"}, {"missing.pl0"}, {"-bad"}} {
		var out, diagnostics bytes.Buffer
		if code := run(args, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.Len() == 0 {
			t.Fatalf("%v: %d %q %s", args, code, &out, &diagnostics)
		}
	}
}

func TestCLIRejectSource(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"begin\nwrite 1", "2:8: unexpected"},
		{"begin write 1; write @ end.", "1:22: unexpected character"},
		{"begin write 1; call missing end.", "undefined"},
	} {
		filename := filepath.Join(t.TempDir(), "invalid.pl0")
		if err := os.WriteFile(filename, []byte(tc.source), 0600); err != nil {
			t.Fatal(err)
		}
		for _, frontend := range frontends {
			t.Run(frontend.name+"/"+tc.source, func(t *testing.T) {
				var out, diagnostics bytes.Buffer
				code := run([]string{"-parser", frontend.name, filename}, &out, &diagnostics)
				if code != 1 || out.Len() != 0 || !strings.Contains(diagnostics.String(), filename+": ") || !strings.Contains(diagnostics.String(), tc.want) {
					t.Fatalf("got %d stdout %q stderr %q; want no output and %q", code, &out, &diagnostics, tc.want)
				}
			})
		}
	}
}

// Execute every shell block, including the embedded complete caller, in a
// disposable source copy so regeneration never modifies the working checkout.
func TestPL0Tutorials(t *testing.T) {
	for _, frontend := range []string{"glcc", "glime", "recursive"} {
		t.Run(frontend, func(t *testing.T) {
			doc, err := os.ReadFile("../../docs/tutorials/pl0-" + frontend + ".md")
			if err != nil {
				t.Fatal(err)
			}
			var commands strings.Builder
			inShell := false
			for line := range strings.SplitSeq(string(doc), "\n") {
				if line == "```sh" {
					inShell = true
					continue
				}
				if line == "```" {
					inShell = false
				}
				if inShell {
					commands.WriteString(line + "\n")
				}
			}
			if commands.Len() == 0 {
				t.Fatal("tutorial has no shell commands")
			}
			dir := t.TempDir()
			for _, name := range []string{"cmd", "internal", "reference", "examples"} {
				if err := os.CopyFS(filepath.Join(dir, name), os.DirFS(filepath.Join("../..", name))); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"go.mod", "go.sum"} {
				data, err := os.ReadFile(filepath.Join("../..", name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("sh", "-c", commands.String())
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "CGO_ENABLED=0", "GOTOOLCHAIN=local")
			output, err := cmd.CombinedOutput()
			if want := "1\n4\n9\n16\n25\n55\nAST: 1 procedure(s), 4 statements\n10\n"; err != nil || string(output) != want {
				t.Fatalf("tutorial: %v\n%s\nwant:\n%s", err, output, want)
			}
		})
	}
}
