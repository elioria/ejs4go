package ejs4go

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFromFileWithDiskIncludes exercises the filesystem path end to end:
// FromFile + FileLoader resolving an include relative to the template file.
func TestFromFileWithDiskIncludes(t *testing.T) {
	dir := t.TempDir()

	// Write a layout that includes a sibling partial by relative name.
	layout := filepath.Join(dir, "page.ejs")
	if err := os.WriteFile(layout, []byte(
		`<h1><%= title %></h1>
<%- include('partial', { name: title }) %>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "partial.ejs"), []byte(
		`<p>hello <%= name.toUpperCase() %></p>`), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := RenderFile(layout, map[string]any{"title": "Docs"},
		WithLoader(NewFileLoader(dir)))
	if err != nil {
		t.Fatalf("RenderFile: %v", err)
	}
	want := "<h1>Docs</h1>\n<p>hello DOCS</p>"
	if out != want {
		t.Errorf("file render:\n got %q\nwant %q", out, want)
	}
}

// TestFromFileMissing returns an error rather than panicking.
func TestFromFileMissing(t *testing.T) {
	_, err := FromFile(filepath.Join(t.TempDir(), "nope.ejs"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestFileLoaderExtAppend verifies the ".ejs" extension is appended.
func TestFileLoaderExtAppend(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "frag.ejs"), []byte("X<%= n %>X"), 0o600); err != nil {
		t.Fatal(err)
	}
	l := NewFileLoader(dir)
	src, resolved, err := l.Load("", "frag") // no extension supplied
	if err != nil {
		t.Fatal(err)
	}
	if src != "X<%= n %>X" {
		t.Errorf("loaded %q", src)
	}
	if filepath.Ext(resolved) != ".ejs" {
		t.Errorf("resolved without .ejs: %q", resolved)
	}
}

// TestDebugMode exercises the compileDebug try/catch wrapper path: a runtime
// error inside the body is caught and rethrown as a RuntimeError.
func TestDebugMode(t *testing.T) {
	// Valid render still works with Debug on.
	out, err := Render("ok <%= 1 + 1 %>", nil, WithDebug(true))
	if err != nil {
		t.Fatalf("debug valid render: %v", err)
	}
	if out != "ok 2" {
		t.Errorf("debug render got %q", out)
	}

	// Error inside body is surfaced (the __rethrow path re-throws).
	_, err = Render("<% throw new Error('boom') %>", nil, WithDebug(true))
	if err == nil {
		t.Fatal("expected error from thrown exception under debug")
	}
}

// TestSourceExposed gives non-zero coverage to Template.Source and documents
// that the generated JS is inspectable.
func TestSourceExposed(t *testing.T) {
	tmpl, err := FromString("<%= x %>")
	if err != nil {
		t.Fatal(err)
	}
	src := tmpl.Source()
	if src == "" {
		t.Fatal("empty source")
	}
	for _, frag := range []string{"__output", "__append", "escapeFn", "return __output"} {
		if !contains(src, frag) {
			t.Errorf("source missing %q:\n%s", frag, src)
		}
	}
}

// TestControlCharsInText covers jsString's escaping of tab, CR, and other
// control characters embedded in literal template text.
func TestControlCharsInText(t *testing.T) {
	// Literal text containing a tab, a carriage return, and a NUL byte.
	src := "a\tb\rc\x01d<%= 1 %>"
	out, err := Render(src, nil)
	if err != nil {
		t.Fatalf("control chars: %v", err)
	}
	want := "a\tb\rc\x01d1"
	if out != want {
		t.Errorf("control chars: got %q want %q", out, want)
	}
}

// TestRenderFileError covers the error return of RenderFile (missing file).
func TestRenderFileError(t *testing.T) {
	_, err := RenderFile(filepath.Join(t.TempDir(), "absent.ejs"), nil)
	if err == nil {
		t.Fatal("expected error from RenderFile on missing file")
	}
}

// TestIncludeDataNonMap drives the include data-coercion path when the data
// object exported from JS is not directly a map[string]any (e.g. built from a
// JS expression). The include should still receive usable locals.
func TestIncludeDataNonMap(t *testing.T) {
	loader := NewMapLoader(map[string]string{
		"greet.ejs": "Hi <%= who %>",
	})
	// Build the include data inline in JS via an object literal.
	out, err := Render(
		"<%- include('greet', Object.assign({}, { who: name })) %>",
		map[string]any{"name": "Eve"},
		WithLoader(loader),
	)
	if err != nil {
		t.Fatalf("non-map include data: %v", err)
	}
	if out != "Hi Eve" {
		t.Errorf("got %q", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
