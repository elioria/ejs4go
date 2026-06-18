package ejs4go

import "strings"

// Options controls how templates are parsed, compiled, and rendered.
//
// The zero value is not usable directly; use DefaultOptions or build via
// functional Option values passed to New/FromString/FromFile.
type Options struct {
	// Delimiter is the inner character of a tag (default "%").
	Delimiter string
	// OpenDelimiter is the opening character of a tag (default "<").
	OpenDelimiter string
	// CloseDelimiter is the closing character of a tag (default ">").
	CloseDelimiter string

	// LocalsName is the name of the object holding template locals when
	// _with is disabled (default "locals").
	LocalsName string
	// With, when true (the default), wraps the template body in
	// `with (locals || {}) { ... }` so locals are accessible by bare name.
	With bool

	// RmWhitespace removes all safe-to-remove whitespace, including leading
	// and trailing whitespace on a line, and enables line-slurping with -%>.
	RmWhitespace bool

	// Strict compiles the generated function in strict mode. Strict mode
	// forbids `with`, so With is implicitly treated as false when Strict.
	Strict bool

	// Debug, when true, wraps the template body in try/catch and rethrows
	// errors annotated with the offending source line.
	Debug bool

	// EscapeFunc is applied to values emitted by <%= %>. Defaults to
	// HTMLEscape. Override to customize (or to no-op) escaping.
	EscapeFunc func(string) string

	// Loader resolves include() paths to template source. Defaults to a
	// filesystem loader rooted at the including template's directory.
	Loader Loader

	// Filename is the logical name of the template, used for error messages
	// and as the base for resolving relative includes.
	Filename string
}

// Option mutates an Options value. Passed variadically to constructors.
type Option func(*Options)

// DefaultOptions returns Options matching EJS defaults.
func DefaultOptions() Options {
	return Options{
		Delimiter:      "%",
		OpenDelimiter:  "<",
		CloseDelimiter: ">",
		LocalsName:     "locals",
		With:           true,
		EscapeFunc:     HTMLEscape,
	}
}

// normalize fills in empty fields with defaults and resolves implied settings.
func (o *Options) normalize() {
	if o.Delimiter == "" {
		o.Delimiter = "%"
	}
	if o.OpenDelimiter == "" {
		o.OpenDelimiter = "<"
	}
	if o.CloseDelimiter == "" {
		o.CloseDelimiter = ">"
	}
	if o.LocalsName == "" {
		o.LocalsName = "locals"
	}
	if o.EscapeFunc == nil {
		o.EscapeFunc = HTMLEscape
	}
	if o.Strict {
		// `with` is illegal in strict mode.
		o.With = false
	}
}

// open returns the opening tag prefix, e.g. "<%".
func (o *Options) open() string { return o.OpenDelimiter + o.Delimiter }

// close returns the closing tag suffix, e.g. "%>".
func (o *Options) close() string { return o.Delimiter + o.CloseDelimiter }

// --- Functional options ---

// WithDelimiter sets the inner delimiter character (default "%").
func WithDelimiter(d string) Option { return func(o *Options) { o.Delimiter = d } }

// WithOpenDelimiter sets the opening delimiter character (default "<").
func WithOpenDelimiter(d string) Option { return func(o *Options) { o.OpenDelimiter = d } }

// WithCloseDelimiter sets the closing delimiter character (default ">").
func WithCloseDelimiter(d string) Option { return func(o *Options) { o.CloseDelimiter = d } }

// WithLocalsName sets the locals object name used when With is disabled.
func WithLocalsName(n string) Option { return func(o *Options) { o.LocalsName = n } }

// WithWith toggles the `with (locals)` wrapper (default true).
func WithWith(b bool) Option { return func(o *Options) { o.With = b } }

// WithRmWhitespace toggles aggressive whitespace removal.
func WithRmWhitespace(b bool) Option { return func(o *Options) { o.RmWhitespace = b } }

// WithStrict toggles strict-mode compilation (implies With=false).
func WithStrict(b bool) Option { return func(o *Options) { o.Strict = b } }

// WithDebug toggles try/catch error annotation.
func WithDebug(b bool) Option { return func(o *Options) { o.Debug = b } }

// WithEscapeFunc overrides the escape function used by <%= %>.
func WithEscapeFunc(fn func(string) string) Option { return func(o *Options) { o.EscapeFunc = fn } }

// WithLoader sets the include() loader.
func WithLoader(l Loader) Option { return func(o *Options) { o.Loader = l } }

// WithFilename sets the logical template filename.
func WithFilename(name string) Option { return func(o *Options) { o.Filename = name } }

// HTMLEscape escapes the five HTML-significant characters, matching EJS's
// default escape function (which escapes &, <, >, ", and ').
func HTMLEscape(s string) string {
	// Replicate ejs/utils.escapeXML ordering: & first to avoid double-escaping.
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&#34;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}
