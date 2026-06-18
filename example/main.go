// Command example is a Gin REST API that renders EJS templates through the
// ejs4go package. Each route exercises a distinct EJS capability; the
// /dashboard route combines them all into one large template.
//
// Run it:
//
//	go run ./example
//	curl -s localhost:8080/render/escaped -d '{"message":"<b>hi</b>"}'
package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/elioria/ejs4go"
)

// loader backs all include() calls with the in-memory partials map.
var loader = ejs4go.NewMapLoader(partials)

// renderText is a small helper: render src with locals and write text/html or
// a JSON error. It centralizes the error-to-HTTP mapping so every endpoint
// reports syntax/runtime failures consistently.
func renderText(c *gin.Context, src string, locals map[string]any, opts ...ejs4go.Option) {
	opts = append(opts, ejs4go.WithLoader(loader))
	out, err := ejs4go.Render(src, locals, opts...)
	if err != nil {
		// Distinguish error classes for the client.
		status := http.StatusInternalServerError
		kind := "runtime_error"
		switch err.(type) {
		case *ejs4go.SyntaxError:
			status = http.StatusBadRequest
			kind = "syntax_error"
		case *ejs4go.CompileError:
			status = http.StatusBadRequest
			kind = "compile_error"
		case *ejs4go.RuntimeError:
			status = http.StatusUnprocessableEntity
			kind = "runtime_error"
		}
		c.JSON(status, gin.H{"error": kind, "detail": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(out))
}

// bindLocals reads a JSON object body into a locals map. A missing/empty body
// yields an empty map rather than an error, so endpoints can supply defaults.
func bindLocals(c *gin.Context) map[string]any {
	locals := map[string]any{}
	// Ignore EOF/empty-body errors; only surface malformed JSON.
	if c.Request.ContentLength != 0 {
		_ = c.ShouldBindJSON(&locals)
	}
	return locals
}

// withDefault returns locals[key] or def when absent.
func withDefault(locals map[string]any, key string, def any) any {
	if v, ok := locals[key]; ok {
		return v
	}
	locals[key] = def
	return def
}

// NewRouter builds the Gin engine with all routes registered. Exposed so the
// integration tests can drive it without binding a port.
func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health/index.
	r.GET("/", func(c *gin.Context) {
		renderText(c, tmplText, nil)
	})

	api := r.Group("/render")
	{
		// <%= %> escaped output.
		api.POST("/escaped", func(c *gin.Context) {
			locals := bindLocals(c)
			withDefault(locals, "message", "hello & welcome")
			renderText(c, tmplEscaped, locals)
		})

		// <%- %> raw output.
		api.POST("/raw", func(c *gin.Context) {
			locals := bindLocals(c)
			withDefault(locals, "html", "<em>raw</em>")
			renderText(c, tmplRaw, locals)
		})

		// <% %> eval / control flow.
		api.POST("/eval", func(c *gin.Context) {
			locals := bindLocals(c)
			withDefault(locals, "n", 3)
			renderText(c, tmplEval, locals)
		})

		// <%# %> comment.
		api.GET("/comment", func(c *gin.Context) {
			renderText(c, tmplComment, nil)
		})

		// <%% %%> literal delimiters.
		api.GET("/literal", func(c *gin.Context) {
			renderText(c, tmplLiteral, nil)
		})

		// -%> newline trim.
		api.POST("/trim", func(c *gin.Context) {
			locals := bindLocals(c)
			withDefault(locals, "items", []any{"a", "b", "c"})
			renderText(c, tmplTrim, locals)
		})

		// <%_ _%> whitespace slurp.
		api.GET("/slurp", func(c *gin.Context) {
			renderText(c, tmplSlurp, nil)
		})

		// rmWhitespace option.
		api.GET("/rmwhitespace", func(c *gin.Context) {
			renderText(c, tmplRmWhitespace, nil, ejs4go.WithRmWhitespace(true))
		})

		// custom delimiters [%  %].
		api.POST("/custom-delim", func(c *gin.Context) {
			locals := bindLocals(c)
			withDefault(locals, "name", "Sam")
			withDefault(locals, "count", 5)
			renderText(c, tmplCustomDelim, locals,
				ejs4go.WithOpenDelimiter("["), ejs4go.WithCloseDelimiter("]"))
		})

		// strict mode (no `with`).
		api.POST("/strict", func(c *gin.Context) {
			locals := bindLocals(c)
			withDefault(locals, "greeting", "Hi")
			withDefault(locals, "who", "there")
			renderText(c, tmplStrict, locals, ejs4go.WithStrict(true))
		})

		// include() single.
		api.POST("/include", func(c *gin.Context) {
			locals := bindLocals(c)
			withDefault(locals, "status", "online")
			renderText(c, tmplInclude, locals)
		})

		// nested include() (partial includes a partial, inside a loop).
		api.POST("/nested-include", func(c *gin.Context) {
			locals := bindLocals(c)
			if _, ok := locals["users"]; !ok {
				locals["users"] = demoUsers()
			}
			renderText(c, tmplNestedInclude, locals)
		})

		// broad real-JavaScript exercise.
		api.POST("/js", func(c *gin.Context) {
			locals := bindLocals(c)
			if _, ok := locals["data"]; !ok {
				locals["data"] = map[string]any{
					"numbers": []any{1, 2, 3, 4, 5},
					"word":    "ejs",
				}
			}
			renderText(c, tmplRealJS, locals)
		})

		// syntax error path.
		api.GET("/bad-syntax", func(c *gin.Context) {
			renderText(c, tmplBadSyntax, nil)
		})

		// runtime error path.
		api.GET("/bad-runtime", func(c *gin.Context) {
			renderText(c, tmplBadRuntime, nil)
		})
	}

	// The big combined template.
	r.POST("/dashboard", func(c *gin.Context) {
		locals := bindLocals(c)
		withDefault(locals, "title", "EJS4GO Dashboard")
		withDefault(locals, "owner", "ops-team")
		withDefault(locals, "year", 2026)
		withDefault(locals, "note", "<b>be careful</b>")
		if _, ok := locals["users"]; !ok {
			locals["users"] = demoUsers()
		}
		renderText(c, tmplDashboard, locals)
	})

	return r
}

// demoUsers is the default dataset for table/dashboard endpoints.
func demoUsers() []any {
	return []any{
		map[string]any{"name": "Alice", "score": 92, "active": true},
		map[string]any{"name": "Bob <x>", "score": 47, "active": false},
		map[string]any{"name": "Cara", "score": 73, "active": true},
	}
}

func main() {
	r := NewRouter()
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	// nosemgrep: listen on all interfaces is intentional for a local demo.
	_ = r.Run(addr)
}
