package lemon_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mdhender/parser/internal/lemon"
)

// The checked-in expectations come only from pinned C Lemon. Ordinary tests
// neither execute C nor download sources, and exercise the public command API.
func TestGrammarOracle(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "testdata/grammar/cases.tsv"))
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
				path := filepath.Join(root, "testdata/grammar/fixtures", input)
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
				want, err := os.ReadFile(filepath.Join(root, "testdata/grammar/golden", name, result.name))
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

func TestGenerationNotYetImplemented(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("grammar.y", []byte("start ::= T.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	if status := lemon.Run([]string{"lemon", "grammar.y"}, &out, &errout); status != 1 || out.Len() != 0 || !strings.Contains(errout.String(), "C parser generation is not implemented") {
		t.Fatalf("status=%d stdout=%s stderr=%s", status, &out, &errout)
	}
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatal("unsupported generation created artifacts")
	}
}
