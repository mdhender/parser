package lemon_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mdhender/parser/internal/lemon"
)

// The checked-in expectations come only from pinned C Lemon. Ordinary tests
// neither execute C nor download sources, and exercise the public command API.
func TestGrammarOracle(t *testing.T) {
	testOracle(t, "grammar")
}

func TestAutomatonOracle(t *testing.T) {
	testOracle(t, "automaton")
}

func TestTableOracle(t *testing.T) {
	testOracle(t, "tables")
}

func testOracle(t *testing.T, stage string) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "testdata", stage, "cases.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(manifest), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 || len(fields) > 4 {
			t.Fatalf("invalid case: %q", line)
		}
		name, operation, input := fields[0], fields[1], fields[2]
		t.Run(name, func(t *testing.T) {
			args := []string{"./lemon"}
			switch operation {
			case "tables":
				args = append(args, "--table-snapshot")
			case "automaton":
				args = append(args, "--automaton-snapshot")
			case "snapshot":
				args = append(args, "--grammar-snapshot")
			case "reprint":
				args = append(args, "-g")
			case "preprocess":
				args = append(args, "-E")
			default:
				t.Fatalf("unknown operation %q", operation)
			}
			if len(fields) == 4 {
				args = append(args, strings.Fields(fields[3])...)
			}
			args = append(args, "grammar.y")
			t.Chdir(t.TempDir())
			if input != "missing" {
				path := filepath.Join(root, "testdata", stage, "fixtures", input)
				if stage == "tables" {
					path = filepath.Join(root, "testdata", stage, input)
				}
				if input == "sqlite" {
					path = filepath.Join(root, "reference/sqlite/src/parse.y")
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile("grammar.y", data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var stdout, stderr bytes.Buffer
			status := lemon.Run(args, &stdout, &stderr)
			for _, result := range []struct {
				name string
				got  []byte
			}{
				{"stdout.txt", stdout.Bytes()}, {"stderr.txt", stderr.Bytes()}, {"exit-status.txt", []byte(strconv.Itoa(status) + "\n")},
			} {
				want, err := os.ReadFile(filepath.Join(root, "testdata", stage, "golden", name, result.name))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(result.got, want) {
					i := 0
					for i < len(want) && i < len(result.got) && want[i] == result.got[i] {
						i++
					}
					t.Errorf("%s differs at byte %d (line %d): got %d bytes, want %d", result.name, i, bytes.Count(want[:i], []byte{'\n'})+1, len(result.got), len(want))
				}
			}
			files, err := os.ReadDir(".")
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range files {
				if f.Name() != "grammar.y" {
					t.Errorf("unexpected generated artifact %s", f.Name())
				}
			}
		})
	}
}

func TestRHSLimit(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, n := range []int{1000, 1001} {
		input := "start ::=" + strings.Repeat(" T", n) + ".\n"
		if err := os.WriteFile("grammar.y", []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		var out, errout bytes.Buffer
		status := lemon.Run([]string{"lemon", "-g", "grammar.y"}, &out, &errout)
		if n == 1000 {
			if status != 0 || errout.Len() != 0 || !strings.HasSuffix(out.String(), input) {
				t.Fatalf("1000 RHS symbols: status=%d stderr=%s", status, &errout)
			}
		} else if status != 1 || out.Len() != 0 || errout.String() != "grammar.y:1: Too many symbols on RHS of rule beginning at \"T\".\n" {
			t.Fatalf("1001 RHS symbols: status=%d stderr=%s", status, &errout)
		}
	}
}

func TestGenerationOracle(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "testdata/generation/cases.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	template, err := os.ReadFile(filepath.Join(root, "reference/sqlite/tool/lempar.c"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(manifest), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		name, operation, input := fields[0], fields[1], fields[2]
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			read := func(path string) []byte {
				t.Helper()
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				return b
			}
			write := func(path string, data []byte) {
				t.Helper()
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Mkdir("out", 0700); err != nil {
				t.Fatal(err)
			}
			filename, argv0 := "grammar.y", "./lemon"
			if operation == "baseline" {
				filename = "parse.y"
			}
			if input != "missing" {
				path := filepath.Join(root, "testdata/generation", input)
				if input == "sqlite" {
					path = filepath.Join(root, "reference/sqlite/src/parse.y")
				}
				write(filename, read(path))
			}
			templateName := "lempar.c"
			if operation == "adjacent" {
				templateName = "grammar.lt"
			}
			if operation == "executable" {
				if err := os.Mkdir("bin", 0700); err != nil {
					t.Fatal(err)
				}
				templateName = "bin/lempar.c"
				argv0 = "./bin/lemon"
			}
			write(templateName, template)
			args := []string{argv0}
			if len(fields) == 4 {
				args = append(args, strings.Fields(fields[3])...)
			}
			args = append(args, filename)
			var out, errout bytes.Buffer
			status := lemon.Run(args, &out, &errout)
			write("stdout.txt", out.Bytes())
			write("stderr.txt", errout.Bytes())
			write("exit-status.txt", []byte(strconv.Itoa(status)+"\n"))
			var hashes strings.Builder
			err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || path == filename || path == templateName {
					return nil
				}
				fmt.Fprintf(&hashes, "%x  ./%s\n", sha256.Sum256(read(path)), filepath.ToSlash(path))
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			want := read(filepath.Join(root, "testdata/generation/golden", name+".sha256"))
			if hashes.String() != string(want) {
				t.Errorf("artifact inventory/hashes differ:\ngot:\n%swant:\n%s", hashes.String(), want)
			}
			if name == "minimal" {
				// An unchanged header must not be rewritten, but a different
				// header must be replaced. Use an old timestamp, not sleeps.
				old := time.Unix(123456789, 0)
				if err := os.Chtimes("grammar.h", old, old); err != nil {
					t.Fatal(err)
				}
				out.Reset()
				errout.Reset()
				if lemon.Run(args, &out, &errout) != 0 {
					t.Fatal(&errout)
				}
				info, err := os.Stat("grammar.h")
				if err != nil {
					t.Fatal(err)
				}
				if !info.ModTime().Equal(old) {
					t.Error("unchanged header rewritten")
				}
				header := read("grammar.h")
				write("grammar.h", []byte("stale\n"))
				if lemon.Run(args, &out, &errout) != 0 || !bytes.Equal(read("grammar.h"), header) {
					t.Error("stale header not replaced")
				}
			}
		})
	}
}
