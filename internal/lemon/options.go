package lemon

import (
	"fmt"
	"io"
	"strings"
)

// Keep the pinned main.options table order: it is observable in diagnostics.
var optionTable = []struct {
	label     byte
	stringArg bool
	help      string
}{
	{'b', false, "Print only the basis in report."},
	{'c', false, "Don't compress the action table."},
	{'d', true, "Output directory.  Default '.'"},
	{'D', true, "Define an %ifdef macro."},
	{'E', false, "Print input file after preprocessing."},
	{'f', true, "Ignored.  (Placeholder for -f compiler options.)"},
	{'g', false, "Print grammar without actions."},
	{'I', true, "Ignored.  (Placeholder for '-I' compiler options.)"},
	{'m', false, "Output a makeheaders compatible file."},
	{'l', false, "Do not print #line statements."},
	{'O', true, "Ignored.  (Placeholder for '-O' compiler options.)"},
	{'p', false, "Show conflicts resolved by precedence rules"},
	{'q', false, "(Quiet) Don't print the report file."},
	{'r', false, "Do not sort or renumber states"},
	{'s', false, "Print parser stats to standard output."},
	{'S', false, "Generate the *.sql file describing the parser tables."},
	{'x', false, "Print the version number."},
	{'T', true, "Specify a template file."},
	{'U', true, "Undefine a macro."},
	{'W', true, "Ignored.  (Placeholder for '-W' compiler options.)"},
}

type options struct {
	flags          map[byte]bool
	defines, files []string
	snapshot       bool
	automaton      bool
	tables         bool
}

func parseOptions(args []string, stderr io.Writer) (options, bool) {
	o := options{flags: make(map[byte]bool)}
	errors := 0
	for i, a := range args[1:] {
		if a == "--grammar-snapshot" {
			o.snapshot = true
			continue
		}
		if a == "--automaton-snapshot" {
			o.automaton = true
			continue
		}
		if a == "--table-snapshot" {
			o.tables = true
			continue
		}
		flag := len(a) > 0 && (a[0] == '-' || a[0] == '+')
		eq := strings.IndexByte(a, '=')
		if !flag && eq < 0 {
			o.files = append(o.files, a)
			continue
		}
		label, value := byte(0), ""
		if flag && len(a) > 1 {
			label = a[1]
			value = a[2:]
		} else if !flag && eq == 1 {
			label = a[0]
			value = a[2:]
		}
		found, isString := false, false
		for _, opt := range optionTable {
			if opt.label == label {
				found = true
				isString = opt.stringArg
				break
			}
		}
		message := ""
		if !found {
			message = "undefined option."
		} else if !flag && !isString {
			message = "option requires an argument."
		}
		if message != "" {
			fmt.Fprintf(stderr, "Command line syntax error: %s\n", message)
			fmt.Fprintln(stderr, strings.Join(args, " "))
			count := len(args[0]) + 1
			for _, prev := range args[1 : i+1] {
				count += len(prev) + 1
			}
			if flag {
				count++
			}
			if count < 20 {
				fmt.Fprintf(stderr, "%*s^-- here\n", count, "")
			} else {
				fmt.Fprintf(stderr, "%*shere --^\n", count-7, "")
			}
			errors++
			continue
		}
		if !isString {
			o.flags[label] = a[0] == '-'
			continue
		}
		switch label {
		case 'D':
			if j := strings.IndexByte(value, '='); j >= 0 {
				value = value[:j]
			}
			o.defines = append(o.defines, value)
		case 'U':
			for j, d := range o.defines {
				if d == value {
					o.defines[j] = o.defines[len(o.defines)-1]
					o.defines = o.defines[:len(o.defines)-1]
					break
				}
			}
		}
	}
	if errors > 0 {
		fmt.Fprintf(stderr, "Valid command line options for \"%s\" are:\n", args[0])
		for _, opt := range optionTable {
			label := string(opt.label)
			if opt.stringArg {
				label += "<string>"
			}
			fmt.Fprintf(stderr, "  -%-10s  %s\n", label, opt.help)
		}
	}
	return o, errors == 0
}
