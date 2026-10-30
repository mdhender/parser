package reference_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdhender/parser/reference"
)

func TestEmbeddedTemplateHash(t *testing.T) {
	const want = "dc0a1e45bdf5d5413467ff157135d388f552fa90624d9c0b76bee1c55cec427a"
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(reference.ParserTemplate))); got != want {
		t.Fatalf("embedded template SHA-256 = %s, want %s", got, want)
	}
}

// These checks run offline, without compiling or executing the C oracle.
// Changes to pinned inputs or reviewed golden outputs require an explicit
// manifest update, not an automatic regeneration during tests.
func TestPinnedFixtures(t *testing.T) {
	for _, dir := range []string{".", "../testdata/golden/sqlite"} {
		t.Run(dir, func(t *testing.T) {
			manifest, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(manifest)), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) != 2 {
					t.Fatalf("invalid checksum entry: %q", line)
				}
				want, err := hex.DecodeString(fields[0])
				if err != nil || len(want) != sha256.Size {
					t.Fatalf("invalid SHA-256: %q", fields[0])
				}
				t.Run(fields[1], func(t *testing.T) {
					data, err := os.ReadFile(filepath.Join(dir, fields[1]))
					if err != nil {
						t.Fatal(err)
					}
					got := sha256.Sum256(data)
					if hex.EncodeToString(got[:]) != fields[0] {
						t.Errorf("pinned fixture changed: got %x, want %s", got, fields[0])
					}
				})
			}
		})
	}
}
