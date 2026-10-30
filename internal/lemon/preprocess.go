package lemon

import (
	"fmt"
	"strings"
)

// evalBoolean follows eval_preprocessor_boolean, including its early returns
// over the entire remaining expression and permissive unfinished expressions.
// A negative result encodes the syntax-error byte offset, as in C.
func (g *Grammar) evalBoolean(z string) int {
	neg, res, okTerm := false, 0, true
	for i := 0; i < len(z); i++ {
		c := z[i]
		if space(c) {
			continue
		}
		switch {
		case c == '!':
			if !okTerm {
				return -(i + 1)
			}
			neg = !neg
		case strings.HasPrefix(z[i:], "||"):
			if okTerm {
				return -(i + 1)
			}
			if res != 0 {
				return 1
			}
			i++
			okTerm = true
		case strings.HasPrefix(z[i:], "&&"):
			if okTerm {
				return -(i + 1)
			}
			if res == 0 {
				return 0
			}
			i++
			okTerm = true
		case c == '(':
			if !okTerm {
				return -(i + 1)
			}
			n := 1
			for k := i + 1; k < len(z); k++ {
				if z[k] == '(' {
					n++
				}
				if z[k] == ')' {
					n--
					if n == 0 {
						res = g.evalBoolean(z[i+1 : k])
						if res < 0 {
							return -(i - res + 1)
						}
						i = k
						break
					}
				}
			}
			if neg {
				res = 1 - res
				neg = false
			}
			okTerm = false
		case alpha(c):
			if !okTerm {
				return -(i + 1)
			}
			k := i + 1
			for k < len(z) && (alnum(z[k]) || z[k] == '_') {
				k++
			}
			res = 0
			for j, d := range g.defines {
				if d == z[i:k] {
					g.defineUsed[j] = true
					res = 1
					break
				}
			}
			i = k - 1
			if neg {
				res = 1 - res
				neg = false
			}
			okTerm = false
		default:
			return -(i + 1)
		}
	}
	return res
}

func (g *Grammar) preprocess(input []byte) ([]byte, bool) {
	z := append([]byte(nil), input...)
	// C treats the first NUL as EOF, even if fread read more bytes.
	if i := strings.IndexByte(string(z), 0); i >= 0 {
		z = z[:i]
	}
	exclude, start, line, startLine := 0, 0, 1, 1
	blank := func(a, b int) {
		for j := a; j < b; j++ {
			if z[j] != '\n' {
				z[j] = ' '
			}
		}
	}
	for i := 0; i < len(z); i++ {
		if z[i] == '\n' {
			line++
		}
		if z[i] != '%' || (i > 0 && z[i-1] != '\n') {
			continue
		}
		s := string(z[i:])
		end := i
		for end < len(z) && z[end] != '\n' {
			end++
		}
		switch {
		case strings.HasPrefix(s, "%endif") && len(s) > 6 && space(s[6]):
			if exclude > 0 {
				exclude--
				if exclude == 0 {
					blank(start, i)
				}
			}
			blank(i, end)
		case strings.HasPrefix(s, "%else") && len(s) > 5 && space(s[5]):
			if exclude == 1 {
				exclude = 0
				blank(start, i)
			} else if exclude == 0 {
				exclude = 1
				start = i
				startLine = line
			}
			blank(i, end)
		case strings.HasPrefix(s, "%ifdef ") || strings.HasPrefix(s, "%if ") || strings.HasPrefix(s, "%ifndef "):
			if exclude > 0 {
				exclude++
			} else {
				j := i
				for j < len(z) && !space(z[j]) {
					j++
				}
				res := g.evalBoolean(string(z[j:end]))
				if res < 0 {
					fmt.Fprintf(g.stderr, "%%if syntax error on line %d.\n  %s <-- syntax error here\n", line, z[j:j-res])
					return nil, false
				}
				exclude = res
				if j != i+7 {
					exclude = 1 - exclude
				}
				if exclude != 0 {
					start = i
					startLine = line
				}
			}
			blank(i, end)
		}
	}
	if exclude != 0 {
		fmt.Fprintf(g.stderr, "unterminated %%ifdef starting on line %d\n", startLine)
		return nil, false
	}
	return z, true
}
