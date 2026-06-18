// Package ejs4go is a full implementation of the EJS (Embedded JavaScript)
// template language for Go, in the spirit of pongo2 (which implements Jinja2).
//
// Unlike Jinja2, EJS does not define its own expression mini-language: it
// embeds real JavaScript. ejs4go therefore compiles each template into a
// single JavaScript function — exactly as the reference implementation
// (mde/ejs) does — and executes it on the pure-Go ECMAScript engine
// github.com/dop251/goja. This means any valid JavaScript works inside tags:
// arrow functions, array methods, template literals, JSON, etc.
//
// # Tags
//
//	<% code %>     control flow; no output
//	<%= expr %>    output, HTML-escaped
//	<%- expr %>    output, unescaped (raw)
//	<%# comment %> ignored
//	<%% / %%>      literal "<%" / "%>"
//	-%>            trim the following newline
//	<%_  /  _%>    slurp surrounding whitespace
//
// # Quick start
//
//	tmpl, _ := ejs4go.FromString("Hello <%= name.toUpperCase() %>!")
//	out, _ := tmpl.Execute(map[string]any{"name": "world"})
//	// out == "Hello WORLD!"
//
// # Includes
//
//	<%- include('partials/header', { title: title }) %>
//
// Includes are resolved by a Loader (FileLoader, MapLoader, or your own).
// FromFile installs a filesystem loader automatically.
package ejs4go

// Render is a one-shot convenience: parse, compile, and execute src with the
// given locals, returning the rendered output.
func Render(src string, locals map[string]any, opts ...Option) (string, error) {
	tmpl, err := FromString(src, opts...)
	if err != nil {
		return "", err
	}
	return tmpl.Execute(locals)
}

// RenderFile is a one-shot convenience: load, compile, and execute the
// template at filename with the given locals.
func RenderFile(filename string, locals map[string]any, opts ...Option) (string, error) {
	tmpl, err := FromFile(filename, opts...)
	if err != nil {
		return "", err
	}
	return tmpl.Execute(locals)
}
