package ejs4go

import "strings"

// compile turns parsed segments into the body source of a JavaScript function.
//
// The generated function shape mirrors mde/ejs:
//
//	[ "use strict"; ]            // when Strict
//	var __output = "";
//	function __append(s){ if (s !== undefined && s !== null) __output += s; }
//	[ with (locals || {}) { ]   // when With
//	  ; <text/code emitted per segment>
//	[ } ]
//	return __output;
//
// Text segments are emitted as __append("...") with the literal JSON-encoded.
// Escaped/raw expressions are wrapped through __append(escapeFn(...)) or
// __append(...). Eval segments are emitted verbatim as statements.
//
// The function is invoked with a single argument (the locals object) and has
// `escapeFn` and `include` available in its enclosing scope (set as globals
// on the runtime before execution).
func compile(segs []segment, opts *Options) string {
	var b strings.Builder

	if opts.Strict {
		b.WriteString("'use strict';\n")
	}
	b.WriteString("var __output = \"\";\n")
	b.WriteString("function __append(s){ if (s !== undefined && s !== null) __output += s; }\n")

	indent := ""
	if opts.With {
		b.WriteString("with (" + opts.LocalsName + " || {}) {\n")
		indent = "  "
	}

	var body strings.Builder
	for _, s := range segs {
		switch s.mode {
		case modeText:
			if s.text == "" {
				continue
			}
			body.WriteString(indent)
			body.WriteString("; __append(")
			body.WriteString(jsString(s.text))
			body.WriteString(");\n")

		case modeEscaped:
			body.WriteString(indent)
			body.WriteString("; __append(escapeFn(")
			body.WriteString(stripSemi(s.text))
			body.WriteString("));\n")

		case modeRaw:
			body.WriteString(indent)
			body.WriteString("; __append(")
			body.WriteString(stripSemi(s.text))
			body.WriteString(");\n")

		case modeEval:
			body.WriteString(indent)
			body.WriteString("; ")
			body.WriteString(strings.TrimSpace(s.text))
			body.WriteString("\n")

		case modeComment:
			// dropped during parsing; nothing to emit
		}
	}

	if opts.Debug {
		b.WriteString(indent)
		b.WriteString("try {\n")
		b.WriteString(body.String())
		b.WriteString(indent)
		b.WriteString("} catch (e) { __rethrow(e); }\n")
	} else {
		b.WriteString(body.String())
	}

	if opts.With {
		b.WriteString("}\n")
	}

	b.WriteString("return __output;\n")
	return b.String()
}

// wrapFunction wraps a compiled body into a named function expression that
// takes the locals object. The result is assigned to a known global so goja
// can fetch and call it.
func wrapFunction(body, localsName string) string {
	var b strings.Builder
	b.WriteString("(function(")
	b.WriteString(localsName)
	b.WriteString("){\n")
	b.WriteString(body)
	b.WriteString("})")
	return b.String()
}

// stripSemi removes a single trailing semicolon (and surrounding whitespace)
// from an expression so it can be used inside __append(...). EJS does the same
// so that `<%= foo; %>` is valid.
func stripSemi(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ";")
	return strings.TrimSpace(s)
}

// jsString encodes a Go string as a JavaScript string literal (double-quoted),
// escaping control characters and quotes. We avoid encoding/json to keep
// forward slashes and unicode intact and to control escaping precisely.
func jsString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(`\u00`)
				const hex = "0123456789abcdef"
				b.WriteByte(hex[r>>4])
				b.WriteByte(hex[r&0xf])
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
