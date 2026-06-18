package ejs4go

import "fmt"

// SyntaxError is returned when a template cannot be tokenized, e.g. an
// unterminated tag.
type SyntaxError struct {
	Msg  string
	Line int
	File string
}

func (e *SyntaxError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("ejs4go: syntax error in %s:%d: %s", e.File, e.Line, e.Msg)
	}
	return fmt.Sprintf("ejs4go: syntax error at line %d: %s", e.Line, e.Msg)
}

// CompileError wraps a failure to compile the generated JavaScript source.
type CompileError struct {
	Msg  string
	File string
}

func (e *CompileError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("ejs4go: compile error in %s: %s", e.File, e.Msg)
	}
	return fmt.Sprintf("ejs4go: compile error: %s", e.Msg)
}

// RuntimeError wraps a JavaScript exception thrown during rendering.
type RuntimeError struct {
	Msg  string
	File string
}

func (e *RuntimeError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("ejs4go: runtime error in %s: %s", e.File, e.Msg)
	}
	return fmt.Sprintf("ejs4go: runtime error: %s", e.Msg)
}
