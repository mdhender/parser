package lang

import "strconv"

var keywords = map[string]string{
	"const": "CONST", "var": "VAR", "procedure": "PROCEDURE",
	"call": "CALL", "begin": "BEGIN", "end": "END", "if": "IF",
	"then": "THEN", "while": "WHILE", "do": "DO", "odd": "ODD",
	"write": "WRITE",
}

var punctuation = map[string]string{
	":=": "ASSIGN", "<=": "LE", ">=": "GE", "#": "NE",
	"=": "EQ", "<": "LT", ">": "GT", "+": "PLUS", "-": "MINUS",
	"*": "TIMES", "/": "SLASH", "(": "LPAREN", ")": "RPAREN",
	",": "COMMA", ";": "SEMI", ".": "DOT",
}

// Lex scans case-sensitive ASCII PL/0, including // and non-nested /* */
// comments. The final EOF token carries the position after the source.
func Lex(source string) ([]Token, error) {
	var tokens []Token
	i, line, column := 0, 1, 1
	advance := func() {
		if source[i] == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
		i++
	}
	for i < len(source) {
		c := source[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			advance()
			continue
		}
		t := Token{Line: line, Column: column}
		start := i
		if i+1 < len(source) && source[i:i+2] == "//" {
			for i < len(source) && source[i] != '\n' {
				advance()
			}
			continue
		}
		if i+1 < len(source) && source[i:i+2] == "/*" {
			advance()
			advance()
			for i+1 < len(source) && source[i:i+2] != "*/" {
				advance()
			}
			if i+1 >= len(source) {
				return nil, t.Errorf("unterminated comment")
			}
			advance()
			advance()
			continue
		}
		letter := func(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
		digit := func(c byte) bool { return c >= '0' && c <= '9' }
		switch {
		case letter(c):
			for i < len(source) && (letter(source[i]) || digit(source[i])) {
				advance()
			}
			t.Kind = "IDENT"
			if keyword := keywords[source[start:i]]; keyword != "" {
				t.Kind = keyword
			}
		case digit(c):
			for i < len(source) && digit(source[i]) {
				advance()
			}
			t.Kind = "NUMBER"
			var err error
			t.Number, err = strconv.ParseInt(source[start:i], 10, 64)
			if err != nil {
				return nil, t.Errorf("integer literal out of int64 range")
			}
			if i < len(source) && letter(source[i]) {
				return nil, t.Errorf("identifier cannot start with a digit")
			}
		default:
			advance()
			if i < len(source) && punctuation[source[start:i+1]] != "" {
				advance()
			}
			t.Kind = punctuation[source[start:i]]
			if t.Kind == "" {
				return nil, t.Errorf("unexpected character %q", c)
			}
		}
		t.Text = source[start:i]
		tokens = append(tokens, t)
	}
	return append(tokens, Token{Kind: "EOF", Text: "end of input", Line: line, Column: column}), nil
}
