package lemon

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mdhender/parser/reference"
)

func (g *Grammar) fileName(suffix string, o options) string {
	name := g.Filename
	if o.outputDir != nil {
		name = name[strings.LastIndexByte(name, '/')+1:]
		name = *o.outputDir + "/" + name
	}
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name = name[:i]
	}
	return name + suffix
}

func (g *Grammar) openOutput(suffix string, o options) *os.File {
	g.outname = g.fileName(suffix, o)
	f, err := os.Create(g.outname)
	if err != nil {
		fmt.Fprintf(g.stderr, "Can't open file \"%s\".\n", g.outname)
		g.errors++
		return nil
	}
	return f
}

func (g *Grammar) openTemplate(argv0 string, o options) []byte {
	readable := func(name string) bool {
		f, err := os.Open(name)
		if err != nil {
			return false
		}
		f.Close()
		return true
	}
	name := ""
	if o.template != nil {
		name = *o.template
		if !readable(name) {
			fmt.Fprintf(g.stderr, "Can't find the parser driver template file \"%s\".\n", name)
			g.errors++
			return nil
		}
	} else {
		name = g.Filename
		if i := strings.LastIndexByte(name, '.'); i >= 0 {
			name = name[:i]
		}
		name += ".lt"
		if !readable(name) {
			name = "lempar.c"
			if !readable(name) {
				if i := strings.LastIndexByte(argv0, '/'); i >= 0 {
					name = argv0[:i] + "/lempar.c"
				} else {
					path, ok := os.LookupEnv("PATH")
					if !ok {
						path = ".:/bin:/usr/bin"
					}
					for _, dir := range strings.Split(path, ":") {
						name = dir + "/lempar.c"
						if _, err := os.Stat(name); err == nil {
							break
						}
					}
				}
				// Preserve all external lookup choices and their read errors.
				// Only an absent final candidate uses the standalone default.
				if _, err := os.Stat(name); os.IsNotExist(err) {
					return []byte(reference.ParserTemplate)
				}
			}
		}
	}
	b, err := os.ReadFile(name)
	if err != nil {
		fmt.Fprintf(g.stderr, "Can't open the template file \"%s\".\n", name)
		g.errors++
		return nil
	}
	return b
}

// cOutput tracks Lemon's logical line counter, including its historical
// counter quirks. Template input is read in fgets-sized (999-byte) chunks.
type cOutput struct {
	w        io.Writer
	line     int
	template []byte
	name     *string
}

func (e *cOutput) print(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	fmt.Fprint(e.w, s)
	e.line += strings.Count(s, "\n")
}

func (e *cOutput) transfer(skip bool) {
	for len(e.template) > 0 {
		n := min(999, len(e.template))
		if i := strings.IndexByte(string(e.template[:n]), '\n'); i >= 0 {
			n = i + 1
		}
		s := string(e.template[:n])
		e.template = e.template[n:]
		if strings.HasPrefix(s, "%%") {
			return
		}
		e.line++
		if skip {
			continue
		}
		if e.name != nil {
			var b strings.Builder
			for i := 0; i < len(s); {
				if strings.HasPrefix(s[i:], "Parse") && (i == 0 || !alpha(s[i-1])) {
					b.WriteString(*e.name)
					i += 5
				} else {
					b.WriteByte(s[i])
					i++
				}
			}
			s = b.String()
		}
		fmt.Fprint(e.w, s)
	}
}

func (e *cOutput) lineDir(line int, filename string) {
	e.print("#line %d \"%s\"\n", line, strings.ReplaceAll(filename, "\\", "\\\\"))
}

func (g *Grammar) templatePrint(e *cOutput, s *string) {
	if s == nil {
		return
	}
	e.print("%s", *s)
	if !strings.HasSuffix(*s, "\n") {
		e.print("\n")
	}
	if !g.noLineNumbers {
		e.lineDir(e.line+1, g.outname)
	}
}

func minimumSizeType(lower, upper int) (string, int) {
	if lower >= 0 {
		if upper <= 255 {
			return "unsigned char", 1
		}
		if upper < 65535 {
			return "unsigned short int", 2
		}
		return "unsigned int", 4
	}
	if lower >= -127 && upper <= 127 {
		return "signed char", 1
	}
	if lower >= -32767 && upper < 32767 {
		return "short", 2
	}
	return "int", 4
}
