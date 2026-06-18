package ejs4go

import "strings"

// mode classifies a parsed segment of a template.
type mode int

const (
	modeText    mode = iota // literal text between tags
	modeEval                // <% ... %>   run as statement, no output
	modeEscaped             // <%= ... %>  output, HTML-escaped
	modeRaw                 // <%- ... %>  output, unescaped
	modeComment             // <%# ... %>  discarded
)

// Literal delimiters (<%% and %%>) are unescaped directly in the text scan of
// parse() rather than carried as their own segment mode.

// segment is one unit of a parsed template: a run of literal text, or the
// inner code of a tag together with its mode and whitespace-control flags.
type segment struct {
	mode mode
	text string // for modeText: the literal; otherwise the inner code

	// slurpBefore is set by <%_ : strip whitespace preceding the tag.
	slurpBefore bool
	// slurpAfter is set by _%> : strip whitespace following the tag.
	slurpAfter bool
	// trimNewline is set by -%> : remove a single newline following the tag.
	trimNewline bool

	line int // 1-based line number where this segment begins
}

// parse tokenizes src into segments according to the configured delimiters.
//
// It walks the source left to right, alternating between literal text and
// tag bodies. The literal-delimiter sequences <%% and %%> are recognized
// before tag parsing so they can be emitted verbatim.
func parse(src string, opts *Options) ([]segment, error) {
	open := opts.open()         // e.g. "<%"
	closeTag := opts.close()    // e.g. "%>"
	delim := opts.Delimiter     // e.g. "%"
	openCh := opts.OpenDelimiter
	closeCh := opts.CloseDelimiter

	// Literal escapes: "<%%" -> "<%" in output, "%%>" -> "%>" in output.
	litOpen := open + delim       // "<%%"
	litClose := delim + closeTag  // "%%>"

	var segs []segment
	var lit strings.Builder // accumulates literal text
	line := 1
	litLine := 1 // line at which the current literal run started

	flushLiteral := func() {
		if lit.Len() > 0 {
			segs = append(segs, segment{mode: modeText, text: lit.String(), line: litLine})
			lit.Reset()
		}
		litLine = line
	}

	i := 0
	n := len(src)
	for i < n {
		// Literal "<%%": emit a single open delimiter and skip one extra delim char.
		if strings.HasPrefix(src[i:], litOpen) {
			lit.WriteString(open)
			i += len(litOpen)
			continue
		}

		// Literal "%%>": emit a single close delimiter. EJS unescapes the
		// literal close in plain text just as it unescapes the literal open.
		if strings.HasPrefix(src[i:], litClose) {
			lit.WriteString(closeTag)
			i += len(litClose)
			continue
		}

		// Opening tag.
		if strings.HasPrefix(src[i:], open) {
			flushLiteral()
			seg, consumed, err := parseTag(src, i, line, open, closeTag, delim, openCh, closeCh, litClose)
			if err != nil {
				return nil, err
			}
			// advance line counter across the consumed tag span
			line += strings.Count(src[i:i+consumed], "\n")
			i += consumed
			litLine = line
			if seg.mode != modeComment { // comments are dropped entirely
				segs = append(segs, seg)
			}
			continue
		}

		// Ordinary character.
		c := src[i]
		if c == '\n' {
			line++
		}
		lit.WriteByte(c)
		i++
	}
	flushLiteral()

	applyWhitespaceControl(segs, opts)
	return segs, nil
}

// parseTag parses a single tag beginning at src[start] (which starts with the
// open delimiter). It returns the parsed segment and the number of bytes
// consumed (including both delimiters).
func parseTag(src string, start, line int, open, closeTag, delim, openCh, closeCh, litClose string) (segment, int, error) {
	pos := start + len(open) // position just past "<%"
	seg := segment{line: line}

	// Determine mode from the character immediately after the open delimiter.
	if pos < len(src) {
		switch src[pos] {
		case '=':
			seg.mode = modeEscaped
			pos++
		case '-':
			seg.mode = modeRaw
			pos++
		case '#':
			seg.mode = modeComment
			pos++
		case '_':
			seg.slurpBefore = true
			seg.mode = modeEval
			pos++
		default:
			seg.mode = modeEval
		}
	} else {
		seg.mode = modeEval
	}

	// Scan to the matching close delimiter. We recognize, in priority order:
	//   "_%>"  -> slurpAfter close
	//   "-%>"  -> trimNewline close
	//   "%>"   -> normal close
	// Inside the body we must not be fooled by the literal-close "%%>".
	bodyStart := pos
	for pos < len(src) {
		// Skip a literal "%%>" so it does not terminate the tag early.
		if strings.HasPrefix(src[pos:], litClose) {
			pos += len(litClose)
			continue
		}
		// Whitespace-slurp close: "_" + "%>".
		if strings.HasPrefix(src[pos:], "_"+closeTag) {
			seg.slurpAfter = true
			seg.text = src[bodyStart:pos]
			end := pos + len("_"+closeTag)
			return seg, end - start, nil
		}
		// Newline-trim close: "-" + "%>".
		if strings.HasPrefix(src[pos:], "-"+closeTag) {
			seg.trimNewline = true
			seg.text = src[bodyStart:pos]
			end := pos + len("-"+closeTag)
			return seg, end - start, nil
		}
		// Normal close.
		if strings.HasPrefix(src[pos:], closeTag) {
			seg.text = src[bodyStart:pos]
			end := pos + len(closeTag)
			return seg, end - start, nil
		}
		pos++
	}

	return seg, 0, &SyntaxError{
		Msg:  "could not find matching close delimiter " + closeTag,
		Line: line,
	}
}

// applyWhitespaceControl resolves the slurp/trim flags against neighbouring
// text segments, mutating their content in place.
func applyWhitespaceControl(segs []segment, opts *Options) {
	for idx := range segs {
		s := &segs[idx]
		if s.mode == modeText {
			continue
		}
		// slurpBefore (<%_) trims trailing whitespace of the previous text seg.
		if s.slurpBefore && idx > 0 && segs[idx-1].mode == modeText {
			segs[idx-1].text = strings.TrimRight(segs[idx-1].text, " \t\r\n")
		}
		// slurpAfter (_%>) trims leading whitespace of the next text seg.
		if s.slurpAfter && idx+1 < len(segs) && segs[idx+1].mode == modeText {
			segs[idx+1].text = strings.TrimLeft(segs[idx+1].text, " \t\r\n")
		}
		// trimNewline (-%>) removes a single leading newline of the next text seg.
		if s.trimNewline && idx+1 < len(segs) && segs[idx+1].mode == modeText {
			segs[idx+1].text = trimOneNewline(segs[idx+1].text)
		}
	}

	if opts.RmWhitespace {
		for idx := range segs {
			if segs[idx].mode == modeText {
				segs[idx].text = rmWhitespace(segs[idx].text)
			}
		}
	}
}

// trimOneNewline removes a single leading "\n" or "\r\n" if present.
func trimOneNewline(s string) string {
	if strings.HasPrefix(s, "\r\n") {
		return s[2:]
	}
	if strings.HasPrefix(s, "\n") {
		return s[1:]
	}
	return s
}

// rmWhitespace strips leading/trailing whitespace from every line, matching
// EJS's rmWhitespace option behaviour for plain text segments.
func rmWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(strings.TrimLeft(ln, " \t"), " \t\r")
	}
	return strings.Join(lines, "\n")
}
