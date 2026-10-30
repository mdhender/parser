package lemon

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Run executes the currently implemented Lemon command. args includes argv[0].
// It returns the process exit status without exiting, and owns no global state.
func Run(args []string, stdout, stderr io.Writer) int {
	o, ok := parseOptions(args, stderr)
	if !ok {
		return 1
	}
	if o.flags['x'] {
		fmt.Fprintln(stdout, "Lemon version 1.0")
		return 0
	}
	if len(o.files) != 1 {
		fmt.Fprintln(stderr, "Exactly one filename argument is required.")
		return 1
	}
	sort.Strings(o.defines)
	g := &Grammar{Filename: o.files[0], byName: make(map[string]*Symbol), defines: o.defines, defineUsed: make([]bool, len(o.defines)), noLineNumbers: o.flags['l'], stderr: stderr}
	g.symbol("$")
	f, err := os.Open(g.Filename)
	if err != nil {
		g.error(0, "Can't open this file for reading.")
		return 1
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		g.error(0, "Can't open this file for reading.")
		return 1
	}
	if info.Size() > 100000000 {
		g.error(0, "Input file too large.")
		return 1
	}
	input, err := io.ReadAll(io.LimitReader(f, 100000001))
	if err != nil || int64(len(input)) != info.Size() {
		g.error(0, "Can't read in all %d bytes of this file.", info.Size())
		return 1
	}
	z, ok := g.preprocess(input)
	if !ok {
		return 1
	}
	if o.flags['E'] {
		fmt.Fprintf(stdout, "%s\n", z)
		return 0
	}
	g.scan(z)
	if g.errors != 0 {
		return g.errors
	}
	if len(g.Rules) == 0 {
		fmt.Fprintln(stderr, "Empty grammar.")
		return 1
	}
	g.prepare()
	if o.automaton || o.tables {
		g.FindRulePrecedences()
		g.FindFirstSets()
		g.FindStates()
		g.FindLinks()
		g.FindFollowSets()
		g.FindActions()
		if o.tables {
			if !o.flags['c'] {
				g.CompressTables()
			}
			// Like C, -r skips initialization as well as sorting. Do not
			// silently repair its zero retained-state count.
			if !o.flags['r'] {
				g.ResortStates()
			}
			g.TableSnapshot(stdout, g.BuildTables())
		} else {
			g.AutomatonSnapshot(stdout)
		}
		if g.NConflict > 0 {
			fmt.Fprintf(stderr, "%d parsing conflicts.\n", g.NConflict)
		}
		if g.errors > 0 || g.NConflict > 0 {
			return 1
		}
		return 0
	}
	if o.snapshot {
		g.FindRulePrecedences()
		g.Snapshot(stdout)
	} else if o.flags['g'] {
		g.reprint(stdout)
	} else {
		fmt.Fprintln(stderr, "C parser generation is not implemented (Milestone D supports -E, -g, --grammar-snapshot, --automaton-snapshot, and --table-snapshot only).")
		return 1
	}
	if o.flags['s'] {
		fmt.Fprintln(stdout, "Parser statistics:")
		for _, s := range []struct {
			label string
			n     int
		}{
			{"terminal symbols", g.NTerminal}, {"non-terminal symbols", g.NSymbol - g.NTerminal}, {"total symbols", g.NSymbol}, {"rules", len(g.Rules)}, {"states", 0}, {"conflicts", 0}, {"action table entries", 0}, {"lookahead table entries", 0}, {"total table size (bytes)", 0},
		} {
			n := 35 - len(s.label)
			if n > 32 {
				n = 32
			}
			fmt.Fprintf(stdout, "  %s%s %5d\n", s.label, strings.Repeat(".", n), s.n)
		}
	}
	return 0
}
