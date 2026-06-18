# ejs4go Gin example

A [Gin](https://github.com/gin-gonic/gin) REST API that renders EJS templates
through the `ejs4go` package. Every route exercises one EJS capability; the
`/dashboard` route combines them all into one large template.

## Run

```bash
go run ./example
# server on :8080
```

## Endpoints

Each capability has a dedicated endpoint. POST endpoints accept a JSON object
that becomes the template locals (all have sensible defaults if you POST an
empty body).

| Method & path                | EJS capability                       | Try it |
|------------------------------|--------------------------------------|--------|
| `GET  /`                     | plain text passthrough               | `curl -s localhost:8080/` |
| `POST /render/escaped`       | `<%= %>` HTML-escaped output         | `curl -s localhost:8080/render/escaped -d '{"message":"<b>x</b> & y"}'` |
| `POST /render/raw`           | `<%- %>` raw (unescaped) output      | `curl -s localhost:8080/render/raw -d '{"html":"<em>hi</em>"}'` |
| `POST /render/eval`          | `<% %>` control flow (loops)         | `curl -s localhost:8080/render/eval -d '{"n":5}'` |
| `GET  /render/comment`       | `<%# %>` comment (dropped)           | `curl -s localhost:8080/render/comment` |
| `GET  /render/literal`       | `<%% %%>` literal delimiters         | `curl -s localhost:8080/render/literal` |
| `POST /render/trim`          | `-%>` newline trim                   | `curl -s localhost:8080/render/trim -d '{"items":["a","b"]}'` |
| `GET  /render/slurp`         | `<%_  _%>` whitespace slurp          | `curl -s localhost:8080/render/slurp` |
| `GET  /render/rmwhitespace`  | `rmWhitespace` option                | `curl -s localhost:8080/render/rmwhitespace` |
| `POST /render/custom-delim`  | custom `[%  %]` delimiters           | `curl -s localhost:8080/render/custom-delim -d '{"name":"Sam","count":7}'` |
| `POST /render/strict`        | strict mode (no `with`)              | `curl -s localhost:8080/render/strict -d '{"greeting":"Hi","who":"there"}'` |
| `POST /render/include`       | `include()` with data                | `curl -s localhost:8080/render/include -d '{"status":"online"}'` |
| `POST /render/nested-include`| nested `include()` inside a loop     | `curl -s localhost:8080/render/nested-include` |
| `POST /render/js`            | broad real JavaScript (map/filter/reduce, Math, JSON, template-literal-style) | `curl -s localhost:8080/render/js` |
| `GET  /render/bad-syntax`    | syntax-error handling (HTTP 400)     | `curl -s localhost:8080/render/bad-syntax` |
| `GET  /render/bad-runtime`   | runtime-error handling (HTTP 422)    | `curl -s localhost:8080/render/bad-runtime` |
| `POST /dashboard`            | **everything combined**              | `curl -s localhost:8080/dashboard -d '{"title":"Ops"}'` |

## Error responses

Render failures return JSON with an `error` class and `detail`:

```json
{ "error": "syntax_error",  "detail": "ejs4go: syntax error ..." }   // HTTP 400
{ "error": "runtime_error", "detail": "ejs4go: runtime error ..." }  // HTTP 422
```

## Tests

`server_test.go` drives the router in-process with `httptest` and asserts the
rendered output of every endpoint — so the full EJS capability matrix is
verified end-to-end.

```bash
go test ./example/                                   # endpoint tests
go test -coverpkg=github.com/kardec/ejs4go ./...     # coverage of the engine
```
