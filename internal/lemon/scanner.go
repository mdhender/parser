package lemon

// scan is Parse's byte scanner. Delimited tokens retain the opening delimiter
// but omit the closing delimiter. In particular top-level quoted arguments do
// not recognize escapes, unlike quoted literals inside C blocks.
func (g *Grammar) scan(z []byte) {
	p := parser{g: g}
	at := func(i int) byte {
		if i >= len(z) {
			return 0
		}
		return z[i]
	}
	line := 1
	for i := 0; i < len(z); {
		c := z[i]
		if c == '\n' {
			line++
		}
		if space(c) {
			i++
			continue
		}
		if c == '/' && at(i+1) == '/' {
			i += 2
			for at(i) != 0 && at(i) != '\n' {
				i++
			}
			continue
		}
		if c == '/' && at(i+1) == '*' {
			i += 2
			if at(i) == '/' {
				i++
			}
			for at(i) != 0 && (at(i) != '/' || at(i-1) != '*') {
				if at(i) == '\n' {
					line++
				}
				i++
			}
			if at(i) != 0 {
				i++
			}
			continue
		}
		start, tokenLine := i, line
		next := 0
		switch {
		case c == '"':
			i++
			for at(i) != 0 && at(i) != '"' {
				if at(i) == '\n' {
					line++
				}
				i++
			}
			next = i
			if at(i) == 0 {
				// The pinned scanner uses its never-assigned startline (zero).
				g.error(0, "String starting on this line is not terminated before the end of the file.")
			} else {
				next++
			}
		case c == '{':
			i++
			level := 1
			for at(i) != 0 && (level > 1 || at(i) != '}') {
				c = at(i)
				switch {
				case c == '\n':
					line++
				case c == '{':
					level++
				case c == '}':
					level--
				case c == '/' && at(i+1) == '*':
					i += 2
					prev := byte(0)
					for at(i) != 0 && (at(i) != '/' || prev != '*') {
						if at(i) == '\n' {
							line++
						}
						prev = at(i)
						i++
					}
				case c == '/' && at(i+1) == '/':
					i += 2
					for at(i) != 0 && at(i) != '\n' {
						i++
					}
					if at(i) != 0 {
						line++
					}
				case c == '\'' || c == '"':
					quote, prev := c, byte(0)
					i++
					for at(i) != 0 && (at(i) != quote || prev == '\\') {
						if at(i) == '\n' {
							line++
						}
						if prev == '\\' {
							prev = 0
						} else {
							prev = at(i)
						}
						i++
					}
				}
				// Do not reproduce C's read past the terminating NUL inside
				// an unfinished comment/literal in an unfinished code block.
				if at(i) == 0 {
					break
				}
				i++
			}
			next = i
			if at(i) == 0 {
				g.error(tokenLine, "C code starting on this line is not terminated before the end of the file.")
			} else {
				next++
			}
		case alnum(c):
			for alnum(at(i)) || at(i) == '_' {
				i++
			}
			next = i
		case c == ':' && at(i+1) == ':' && at(i+2) == '=':
			i += 3
			next = i
		case (c == '/' || c == '|') && alpha(at(i+1)):
			i += 2
			for alnum(at(i)) || at(i) == '_' {
				i++
			}
			next = i
		default:
			i++
			next = i
		}
		p.token(string(z[start:i]), tokenLine)
		i = next
	}
}
