package ejs4go

import (
	"strings"
	"testing"
)

// TestAllOptionSetters applies every functional option and confirms each
// mutates the resolved Options as documented. This covers the thin setter
// functions and the normalize() defaulting branches.
func TestAllOptionSetters(t *testing.T) {
	o := DefaultOptions()
	custom := func(s string) string { return "[" + s + "]" }
	loader := NewMapLoader(map[string]string{})

	opts := []Option{
		WithDelimiter("@"),
		WithOpenDelimiter("{"),
		WithCloseDelimiter("}"),
		WithLocalsName("ctx"),
		WithWith(false),
		WithRmWhitespace(true),
		WithStrict(false),
		WithDebug(true),
		WithEscapeFunc(custom),
		WithLoader(loader),
		WithFilename("foo.ejs"),
	}
	for _, fn := range opts {
		fn(&o)
	}
	o.normalize()

	if o.Delimiter != "@" || o.OpenDelimiter != "{" || o.CloseDelimiter != "}" {
		t.Errorf("delimiters not set: %+v", o)
	}
	if o.LocalsName != "ctx" {
		t.Errorf("localsName = %q", o.LocalsName)
	}
	if o.With {
		t.Error("With should be false")
	}
	if !o.RmWhitespace || !o.Debug {
		t.Error("rmWhitespace/debug not set")
	}
	if o.EscapeFunc("x") != "[x]" {
		t.Error("escape func not applied")
	}
	if o.Loader == nil || o.Filename != "foo.ejs" {
		t.Error("loader/filename not set")
	}
	if o.open() != "{@" || o.close() != "@}" {
		t.Errorf("open/close = %q/%q", o.open(), o.close())
	}
}

// TestNormalizeDefaults checks empty fields are filled and Strict implies !With.
func TestNormalizeDefaults(t *testing.T) {
	o := Options{Strict: true} // everything else empty
	o.normalize()
	if o.Delimiter != "%" || o.OpenDelimiter != "<" || o.CloseDelimiter != ">" {
		t.Errorf("defaults not applied: %+v", o)
	}
	if o.LocalsName != "locals" {
		t.Errorf("localsName default = %q", o.LocalsName)
	}
	if o.EscapeFunc == nil {
		t.Error("escape func default nil")
	}
	if o.With {
		t.Error("Strict must force With=false")
	}
}

// TestErrorMessages exercises every Error() formatter, with and without a file.
func TestErrorMessages(t *testing.T) {
	cases := []struct {
		err  error
		want []string
	}{
		{&SyntaxError{Msg: "boom", Line: 3}, []string{"syntax error", "line 3", "boom"}},
		{&SyntaxError{Msg: "boom", Line: 3, File: "a.ejs"}, []string{"a.ejs:3", "boom"}},
		{&CompileError{Msg: "bad"}, []string{"compile error", "bad"}},
		{&CompileError{Msg: "bad", File: "b.ejs"}, []string{"b.ejs", "bad"}},
		{&RuntimeError{Msg: "kab"}, []string{"runtime error", "kab"}},
		{&RuntimeError{Msg: "kab", File: "c.ejs"}, []string{"c.ejs", "kab"}},
	}
	for _, tc := range cases {
		msg := tc.err.Error()
		for _, w := range tc.want {
			if !strings.Contains(msg, w) {
				t.Errorf("%T message %q missing %q", tc.err, msg, w)
			}
		}
	}
}

// TestHTMLEscapeAllChars covers every replacement in the default escaper.
func TestHTMLEscapeAllChars(t *testing.T) {
	got := HTMLEscape(`<a href="x" data='y'>&`)
	want := `&lt;a href=&#34;x&#34; data=&#39;y&#39;&gt;&amp;`
	if got != want {
		t.Errorf("escape:\n got %q\nwant %q", got, want)
	}
}

// TestCustomEscapeFuncEndToEnd wires a non-default escape function through a
// real render, covering the escapeFn install path with an override.
func TestCustomEscapeFuncEndToEnd(t *testing.T) {
	out, err := Render("<%= x %>", map[string]any{"x": "ab"},
		WithEscapeFunc(func(s string) string { return strings.ToUpper(s) }))
	if err != nil {
		t.Fatal(err)
	}
	if out != "AB" {
		t.Errorf("custom escape: got %q", out)
	}
}

// TestIncludeWithoutLoaderErrors covers the no-loader include guard.
func TestIncludeWithoutLoaderErrors(t *testing.T) {
	// FromString defaults Loader to nil; include() must error clearly.
	_, err := Render("<%- include('x') %>", nil)
	if err == nil {
		t.Fatal("expected error: include without loader")
	}
	if !strings.Contains(err.Error(), "no Loader") && !strings.Contains(err.Error(), "include") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestNewlineTrimCRLF covers the \r\n branch of trimOneNewline.
func TestNewlineTrimCRLF(t *testing.T) {
	out, err := Render("<% var a=1 -%>\r\nX", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "X" {
		t.Errorf("crlf trim: got %q", out)
	}
}
