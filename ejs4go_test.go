package ejs4go

import (
	"strings"
	"testing"
)

func render(t *testing.T, src string, locals map[string]any, opts ...Option) string {
	t.Helper()
	out, err := Render(src, locals, opts...)
	if err != nil {
		t.Fatalf("Render(%q) error: %v\n--- generated source ---\n%s", src, err, mustSource(src, opts...))
	}
	return out
}

func mustSource(src string, opts ...Option) string {
	tmpl, err := FromString(src, opts...)
	if err != nil {
		return "(compile failed: " + err.Error() + ")"
	}
	return tmpl.Source()
}

func TestPlainText(t *testing.T) {
	if got := render(t, "hello world", nil); got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestEscapedOutput(t *testing.T) {
	got := render(t, "<%= name %>", map[string]any{"name": "<b>x</b>"})
	want := "&lt;b&gt;x&lt;/b&gt;"
	if got != want {
		t.Errorf("escaped: got %q want %q", got, want)
	}
}

func TestRawOutput(t *testing.T) {
	got := render(t, "<%- name %>", map[string]any{"name": "<b>x</b>"})
	if got != "<b>x</b>" {
		t.Errorf("raw: got %q", got)
	}
}

func TestEvalControlFlow(t *testing.T) {
	src := "<% for (var i = 0; i < 3; i++) { %><%= i %><% } %>"
	if got := render(t, src, nil); got != "012" {
		t.Errorf("loop: got %q", got)
	}
}

func TestComment(t *testing.T) {
	if got := render(t, "a<%# this is ignored %>b", nil); got != "ab" {
		t.Errorf("comment: got %q", got)
	}
}

func TestLiteralDelimiter(t *testing.T) {
	if got := render(t, "<%%= raw %>", nil); got != "<%= raw %>" {
		t.Errorf("literal: got %q", got)
	}
}

func TestLiteralCloseInText(t *testing.T) {
	// Both literal open <%% and literal close %%> must unescape in plain text.
	if got := render(t, "use <%%= x %%> here", nil); got != "use <%= x %> here" {
		t.Errorf("literal close: got %q", got)
	}
}

func TestRealJavaScript(t *testing.T) {
	// Exercise ES features goja supports: arrow fns, array methods, template literals.
	src := "<%= [1,2,3].map(function(n){ return n*n; }).join('-') %>"
	if got := render(t, src, nil); got != "1-4-9" {
		t.Errorf("js: got %q", got)
	}
}

func TestExpressionWithTrailingSemicolon(t *testing.T) {
	if got := render(t, "<%= 1 + 2; %>", nil); got != "3" {
		t.Errorf("semi: got %q", got)
	}
}

func TestUndefinedAppendsNothing(t *testing.T) {
	// __append must drop undefined/null instead of printing "undefined".
	// A *defined* local holding undefined value appends nothing (real EJS:
	// a bare undeclared identifier throws ReferenceError, matching JS).
	got := render(t, "[<%= maybe %>]", map[string]any{"maybe": nil})
	if got != "[]" {
		t.Errorf("undefined: got %q", got)
	}
}

func TestUndeclaredIdentifierThrows(t *testing.T) {
	// Matches reference EJS / JavaScript: referencing an undeclared name
	// inside with(locals) is a ReferenceError.
	_, err := Render("[<%= missing %>]", nil)
	if err == nil {
		t.Fatal("expected ReferenceError for undeclared identifier")
	}
}

func TestNewlineTrim(t *testing.T) {
	src := "<% var x = 1 -%>\nLINE"
	if got := render(t, src, nil); got != "LINE" {
		t.Errorf("trim newline: got %q", got)
	}
}

func TestWhitespaceSlurp(t *testing.T) {
	src := "A   <%_ var y = 1 _%>   B"
	// slurpBefore strips the trailing spaces after A; slurpAfter strips spaces before B.
	if got := render(t, src, nil); got != "AB" {
		t.Errorf("slurp: got %q", got)
	}
}

func TestRmWhitespace(t *testing.T) {
	src := "  <% var z = 1 %>  \n  text  "
	got := render(t, src, nil, WithRmWhitespace(true))
	if strings.Contains(got, "  ") {
		t.Errorf("rmWhitespace left double spaces: got %q", got)
	}
}

func TestCustomDelimiter(t *testing.T) {
	// Coherent custom set: open "[", inner "%", close "]" -> "[%= ... %]".
	src := "Hello [%= name %]!"
	got := render(t, src, map[string]any{"name": "Bob"},
		WithOpenDelimiter("["), WithCloseDelimiter("]"))
	if got != "Hello Bob!" {
		t.Errorf("custom delim: got %q", got)
	}
}

func TestStrictMode(t *testing.T) {
	// In strict mode `with` is disabled; locals must be accessed via localsName.
	src := "<%= locals.name %>"
	got := render(t, src, map[string]any{"name": "Z"}, WithStrict(true))
	if got != "Z" {
		t.Errorf("strict: got %q", got)
	}
}

func TestIncludeViaMapLoader(t *testing.T) {
	loader := NewMapLoader(map[string]string{
		"header.ejs": "<h1><%= title %></h1>",
	})
	src := "<%- include('header', { title: title }) %>!"
	got := render(t, src, map[string]any{"title": "Hi"}, WithLoader(loader))
	if got != "<h1>Hi</h1>!" {
		t.Errorf("include: got %q", got)
	}
}

func TestNestedInclude(t *testing.T) {
	loader := NewMapLoader(map[string]string{
		"outer.ejs": "[<%- include('inner', { v: v }) %>]",
		"inner.ejs": "<%= v * 2 %>",
	})
	out, err := Render("<%- include('outer', { v: 5 }) %>", map[string]any{}, WithLoader(loader))
	if err != nil {
		t.Fatal(err)
	}
	if out != "[10]" {
		t.Errorf("nested include: got %q", out)
	}
}

func TestRuntimeErrorSurfaces(t *testing.T) {
	_, err := Render("<%= notDefined.field %>", nil)
	if err == nil {
		t.Fatal("expected runtime error for property access on undefined")
	}
	if _, ok := err.(*RuntimeError); !ok {
		t.Errorf("expected *RuntimeError, got %T: %v", err, err)
	}
}

func TestSyntaxErrorUnterminated(t *testing.T) {
	_, err := FromString("<%= oops")
	if err == nil {
		t.Fatal("expected syntax error for unterminated tag")
	}
	if _, ok := err.(*SyntaxError); !ok {
		t.Errorf("expected *SyntaxError, got %T: %v", err, err)
	}
}

func TestReuseTemplate(t *testing.T) {
	tmpl, err := FromString("Hi <%= name %>")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := tmpl.Execute(map[string]any{"name": "A"})
	b, _ := tmpl.Execute(map[string]any{"name": "B"})
	if a != "Hi A" || b != "Hi B" {
		t.Errorf("reuse: got %q, %q", a, b)
	}
}
