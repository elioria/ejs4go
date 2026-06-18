# ejs4go

**A full implementation of the [EJS](https://ejs.co) (Embedded JavaScript) template language for Go.**

`ejs4go` is to EJS what [pongo2](https://github.com/flosch/pongo2) is to Jinja2:
a faithful, idiomatic Go port of a popular templating language. The crucial
difference is what EJS *is*. Jinja2 (and pongo2) define their own sandboxed
expression mini-language. **EJS embeds real JavaScript.** So `ejs4go` does not
reimplement an expression evaluator — it compiles each template into a single
JavaScript function and runs it on [goja](https://github.com/dop251/goja), a
pure-Go ECMAScript engine. No cgo, no Node.js, no V8.

```go
out, _ := ejs4go.Render("Hello <%= name.toUpperCase() %>!", map[string]any{"name": "world"})
// "Hello WORLD!"
```

Because the engine runs genuine JavaScript, anything goja supports works inside
your tags: arrow functions, `map`/`filter`/`reduce`, template literals, `JSON`,
`Math`, `Date`, regular expressions, and more.

---

## Why ejs4go?

| | ejs4go | pongo2 |
|---|---|---|
| Language | EJS (real embedded JavaScript) | Jinja2 (sandboxed DSL) |
| Expressions | Full ECMAScript via goja | Custom evaluator |
| Engine | `github.com/dop251/goja` (pure Go) | Built-in |
| cgo / external runtime | None | None |
| Familiar to | Node.js / Express developers | Python / Django developers |

If your team already writes `.ejs` templates for a Node service and you are
moving (or sharing) rendering with a Go backend, `ejs4go` lets the **same
templates** run unchanged.

---

## Install

```bash
go get github.com/elioria/ejs4go
```

Requires Go 1.21+ (uses `map[string]any`).

---

## Tag reference

`ejs4go` implements the complete EJS tag set:

| Tag | Meaning |
|------|---------|
| `<% code %>` | Run JavaScript as a statement; produces **no output** (loops, `if`, variable declarations). |
| `<%= expr %>` | Evaluate `expr` and output it, **HTML-escaped**. |
| `<%- expr %>` | Evaluate `expr` and output it **raw** (no escaping). |
| `<%# comment %>` | A comment. Never evaluated, never rendered. |
| `<%% ... %%>` | Emit a literal `<%` / `%>` (escape the delimiters themselves). |
| `-%>` | Closing tag that **trims the following newline**. |
| `<%_` ... `_%>` | Opening/closing tags that **slurp surrounding whitespace**. |

Delimiters are configurable (see [Options](#options)).

---

## Quick start

### One-shot render

```go
package main

import (
	"fmt"
	"github.com/elioria/ejs4go"
)

func main() {
	out, err := ejs4go.Render(
		"<ul><% items.forEach(function(it){ %><li><%= it %></li><% }) %></ul>",
		map[string]any{"items": []any{"a", "b", "c"}},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
	// <ul><li>a</li><li>b</li><li>c</li></ul>
}
```

### Compile once, render many times

`Render`/`RenderFile` parse and compile on every call. For hot paths, compile a
`*Template` once and reuse it — it is independent of any runtime and safe to
hold for the life of your program.

```go
tmpl, err := ejs4go.FromString("Hi <%= name %>")
if err != nil {
	log.Fatal(err)
}

a, _ := tmpl.Execute(map[string]any{"name": "Alice"}) // "Hi Alice"
b, _ := tmpl.Execute(map[string]any{"name": "Bob"})   // "Hi Bob"
```

> **Concurrency note:** a goja runtime is single-goroutine, so `Execute`
> creates a fresh runtime per call. A single `*Template` is therefore safe to
> call from multiple goroutines concurrently.

---

## Escaping: `<%=` vs `<%-`

```go
data := map[string]any{"comment": "<script>alert(1)</script>"}

ejs4go.Render("<%= comment %>", data)
// &lt;script&gt;alert(1)&lt;/script&gt;   ← escaped, safe

ejs4go.Render("<%- comment %>", data)
// <script>alert(1)</script>             ← raw, only for trusted HTML
```

The default escaper (`HTMLEscape`) escapes `& < > " '`, matching the reference
EJS implementation. Override it with [`WithEscapeFunc`](#options).

---

## Real JavaScript inside templates

Anything goja can run, a template can use:

```go
src := `
<%
  var nums   = data.numbers;
  var total  = nums.reduce(function(a, b){ return a + b; }, 0);
  var evens  = nums.filter(function(n){ return n % 2 === 0; });
-%>
sum=<%= total %>
evens=<%= evens.join(",") %>
max=<%= Math.max.apply(null, nums) %>
json=<%- JSON.stringify({ n: nums.length, total: total }) %>
parity=<%= total % 2 === 0 ? "even" : "odd" %>`

out, _ := ejs4go.Render(src, map[string]any{
	"data": map[string]any{"numbers": []any{1, 2, 3, 4, 5}},
})
// sum=15
// evens=2,4
// max=5
// json={"n":5,"total":15}
// parity=odd
```

---

## Includes (partials)

`include(path, data)` renders another template inline. Includes are resolved by
a **`Loader`**. Three are built in.

### From the filesystem

```go
// templates/page.ejs:    <%- include('partial', { name: title }) %>
// templates/partial.ejs: <p>hello <%= name %></p>

out, _ := ejs4go.RenderFile("templates/page.ejs",
	map[string]any{"title": "Docs"},
	ejs4go.WithLoader(ejs4go.NewFileLoader("templates")),
)
```

`FromFile`/`RenderFile` install a `FileLoader` automatically when none is given,
resolving includes relative to the including template's directory. A `.ejs`
extension is appended when the include name has none.

### From memory (embedded templates, tests)

```go
loader := ejs4go.NewMapLoader(map[string]string{
	"header.ejs": "<h1><%= title %></h1>",
})

out, _ := ejs4go.Render(
	"<%- include('header', { title: title }) %>!",
	map[string]any{"title": "Hi"},
	ejs4go.WithLoader(loader),
)
// <h1>Hi</h1>!
```

Includes nest freely — a partial may include another partial, and includes work
inside loops.

### Custom loaders

Implement the interface to back includes with an `embed.FS`, a database, an
object store, etc.:

```go
type Loader interface {
	Load(from, name string) (src string, resolved string, err error)
}
```

---

## Whitespace control

```go
// -%> removes the newline that follows the tag, keeping output tight:
ejs4go.Render("<% items.forEach(function(i){ -%>\n<%= i %>\n<% }) -%>\nEND",
	map[string]any{"items": []any{"a", "b"}})
// a\nb\nEND

// <%_ and _%> slurp whitespace on each side of the tag:
ejs4go.Render("LEFT   <%_ var x = 1 _%>   RIGHT", nil)
// LEFTRIGHT

// rmWhitespace strips leading/trailing whitespace from every line:
ejs4go.Render("  <% var a = 1 %>\n  <%= a %>  ", nil, ejs4go.WithRmWhitespace(true))
```

---

## Options

Pass functional options to `Render`, `RenderFile`, `FromString`, or `FromFile`.

| Option | Default | Purpose |
|--------|---------|---------|
| `WithOpenDelimiter(s)` | `"<"` | Opening delimiter character. |
| `WithDelimiter(s)` | `"%"` | Inner delimiter character. |
| `WithCloseDelimiter(s)` | `">"` | Closing delimiter character. |
| `WithEscapeFunc(fn)` | `HTMLEscape` | Function applied to `<%= %>` output. |
| `WithLoader(l)` | filesystem | Resolver for `include()`. |
| `WithStrict(b)` | `false` | Compile in strict mode (disables `with`). |
| `WithWith(b)` | `true` | Wrap body in `with (locals)` for bare-name access. |
| `WithLocalsName(s)` | `"locals"` | Name of the locals object when `with` is off. |
| `WithRmWhitespace(b)` | `false` | Aggressive whitespace removal. |
| `WithDebug(b)` | `false` | Wrap body in try/catch and annotate errors. |
| `WithFilename(s)` | `""` | Logical name for errors / relative includes. |

### Custom delimiters

```go
ejs4go.Render("Hello [%= name %]!", map[string]any{"name": "Sam"},
	ejs4go.WithOpenDelimiter("["), ejs4go.WithCloseDelimiter("]"))
// Hello Sam!
```

### Strict mode

Strict mode disables the `with` statement (illegal in strict JS), so locals are
accessed through the locals object by name:

```go
ejs4go.Render("<%= locals.greeting %>, <%= locals.who %>!",
	map[string]any{"greeting": "Hi", "who": "there"},
	ejs4go.WithStrict(true))
// Hi, there!
```

---

## Error handling

Errors are typed so you can react to the failure stage:

```go
_, err := ejs4go.Render(src, locals)
switch e := err.(type) {
case *ejs4go.SyntaxError:   // tokenization failed (e.g. unterminated tag)
	log.Printf("syntax error at line %d: %s", e.Line, e.Msg)
case *ejs4go.CompileError:  // generated JavaScript would not compile
	log.Printf("compile error: %s", e.Msg)
case *ejs4go.RuntimeError:  // a JS exception was thrown while rendering
	log.Printf("runtime error: %s", e.Msg)
case nil:
	// success
default:
	log.Printf("error: %v", err)
}
```

> **EJS / JavaScript semantics:** referencing an *undeclared* identifier inside
> `<%= %>` throws a `ReferenceError` (a `*RuntimeError`), exactly as real EJS and
> JavaScript do. A declared local holding `nil`/`undefined` appends nothing.

---

## API at a glance

```go
// Construct
func FromString(src string, opts ...Option) (*Template, error)
func FromFile(filename string, opts ...Option) (*Template, error)

// Render
func (t *Template) Execute(locals map[string]any) (string, error)
func (t *Template) Source() string   // the generated JS, for debugging

// One-shot convenience
func Render(src string, locals map[string]any, opts ...Option) (string, error)
func RenderFile(filename string, locals map[string]any, opts ...Option) (string, error)

// Loaders
type Loader interface { Load(from, name string) (src, resolved string, err error) }
func NewFileLoader(dir string) *FileLoader
func NewMapLoader(sources map[string]string) *MapLoader

// Escaping
func HTMLEscape(s string) string
```

---

## Example server

The [`example/`](./example) directory contains a complete **Gin REST API** that
renders EJS through this package, with one endpoint per EJS capability plus a
large combined `/dashboard` template. It doubles as an end-to-end test suite
(`example/server_test.go`) covering every feature via `httptest`.

```bash
go run ./example
curl -s localhost:8080/render/escaped -d '{"message":"<b>hi</b> & bye"}'
# <p>&lt;b&gt;hi&lt;/b&gt; &amp; bye</p>
```

See [`example/README.md`](./example/README.md) for the full endpoint table.

---

## How it works

1. **Parse** — the template is tokenized into segments (literal text and tag
   bodies), honoring delimiters and whitespace-control flags.
2. **Compile** — segments are emitted as the body of a single JavaScript
   function that accumulates output in `__output` via `__append(...)`, wrapped
   in `with (locals || {}) { ... }`, and `return`s the string. This mirrors the
   reference implementation, [`mde/ejs`](https://github.com/mde/ejs).
3. **Execute** — the function source is compiled to a goja `*Program` once, then
   invoked with your locals on a fresh runtime per render.

You can inspect the generated JavaScript with `Template.Source()`.

---

## Testing

```bash
go test ./...                                          # all tests
go test -coverpkg=github.com/elioria/ejs4go ./...       # engine coverage
```

The suite covers every EJS capability at both the unit level and through the
example HTTP API.

---

## License

MIT.
# ejs4go
