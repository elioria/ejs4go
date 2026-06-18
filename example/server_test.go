package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// do performs a request against the in-process router and returns status+body.
func do(t *testing.T, method, path, body string) (int, string) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
	}
	w := httptest.NewRecorder()
	NewRouter().ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

// mustContain asserts the body contains every fragment.
func mustContain(t *testing.T, body string, frags ...string) {
	t.Helper()
	for _, f := range frags {
		if !strings.Contains(body, f) {
			t.Errorf("body missing %q\n--- body ---\n%s", f, body)
		}
	}
}

// mustNotContain asserts the body contains none of the fragments.
func mustNotContain(t *testing.T, body string, frags ...string) {
	t.Helper()
	for _, f := range frags {
		if strings.Contains(body, f) {
			t.Errorf("body unexpectedly contains %q\n--- body ---\n%s", f, body)
		}
	}
}

func TestIndexPlainText(t *testing.T) {
	code, body := do(t, http.MethodGet, "/", "")
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	mustContain(t, body, "Welcome to the EJS4GO demo server.")
}

func TestEscaped(t *testing.T) {
	code, body := do(t, http.MethodPost, "/render/escaped", `{"message":"<b>x</b> & y"}`)
	if code != 200 {
		t.Fatalf("status %d body %s", code, body)
	}
	mustContain(t, body, "&lt;b&gt;x&lt;/b&gt; &amp; y")
	mustNotContain(t, body, "<b>x</b>")
}

func TestRaw(t *testing.T) {
	code, body := do(t, http.MethodPost, "/render/raw", `{"html":"<em>raw</em>"}`)
	if code != 200 {
		t.Fatalf("status %d", code)
	}
	mustContain(t, body, "<em>raw</em>")
}

func TestEval(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/eval", `{"n":4}`)
	mustContain(t, body, "Item 1", "Item 2", "Item 3", "Item 4")
	mustNotContain(t, body, "Item 5")
}

func TestComment(t *testing.T) {
	_, body := do(t, http.MethodGet, "/render/comment", "")
	mustContain(t, body, "startend")
	mustNotContain(t, body, "secret note")
}

func TestLiteral(t *testing.T) {
	_, body := do(t, http.MethodGet, "/render/literal", "")
	mustContain(t, body, "<%= value %>")
}

func TestTrim(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/trim", `{"items":["x","y"]}`)
	// -%> trims the newline after each loop control tag, keeping items tight.
	mustContain(t, body, "x\ny\nDONE")
}

func TestSlurp(t *testing.T) {
	_, body := do(t, http.MethodGet, "/render/slurp", "")
	// <%_ strips spaces before tag, _%> strips spaces after -> "LEFTRIGHT".
	mustContain(t, body, "LEFTRIGHT")
}

func TestRmWhitespace(t *testing.T) {
	_, body := do(t, http.MethodGet, "/render/rmwhitespace", "")
	mustContain(t, body, "1")
	if strings.Contains(body, "   ") {
		t.Errorf("rmWhitespace left triple spaces: %q", body)
	}
}

func TestCustomDelim(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/custom-delim", `{"name":"Dana","count":7}`)
	mustContain(t, body, "Hello Dana, you have 7 messages.")
}

func TestStrict(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/strict", `{"greeting":"Hey","who":"world"}`)
	mustContain(t, body, "Hey, world!")
}

func TestInclude(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/include", `{"status":"online"}`)
	mustContain(t, body, `class="badge badge-ok"`, "online")
}

func TestNestedInclude(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/nested-include", "")
	// Default demo users; nested include renders rows + badges.
	mustContain(t, body, "Alice", "badge-ok", "active")
	mustContain(t, body, "Bob &lt;x&gt;") // escaped name in row
	mustContain(t, body, "badge-off", "inactive")
}

func TestRealJS(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/js", "")
	mustContain(t,
		body,
		"doubled=2-4-6-8-10",
		"evens=2-4",
		"total=15",
		"max=5",
		"labels=n1,n2,n3,n4,n5",
		"upper=EJS",
		`json={"count":5,"total":15}`,
		"parity=odd",
	)
}

func TestRealJSCustomData(t *testing.T) {
	body := `{"data":{"numbers":[10,20,30],"word":"go"}}`
	_, out := do(t, http.MethodPost, "/render/js", body)
	mustContain(t, out, "doubled=20-40-60", "total=60", "max=30", "upper=GO", "parity=even")
}

func TestBadSyntax(t *testing.T) {
	code, body := do(t, http.MethodGet, "/render/bad-syntax", "")
	if code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", code)
	}
	var resp map[string]string
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp["error"] != "syntax_error" {
		t.Errorf("want syntax_error, got %q", resp["error"])
	}
}

func TestBadRuntime(t *testing.T) {
	code, body := do(t, http.MethodGet, "/render/bad-runtime", "")
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d (%s)", code, body)
	}
	var resp map[string]string
	_ = json.Unmarshal([]byte(body), &resp)
	if resp["error"] != "runtime_error" {
		t.Errorf("want runtime_error, got %q", resp["error"])
	}
}

func TestDashboardCombinesEverything(t *testing.T) {
	code, body := do(t, http.MethodPost, "/dashboard", `{"title":"Ops","owner":"me","note":"<i>hi</i>"}`)
	if code != 200 {
		t.Fatalf("status %d body %s", code, body)
	}
	mustContain(t, body,
		"<h1>Ops</h1>",            // header include + escaped title
		"generated for me",        // header owner
		"Users: 3 (active: 2)",    // real-JS summary
		"Average score:",          // toFixed
		"Top user: Alice",         // sort + slice
		"Mood: healthy",           // conditional (avg of 92/47/73 = 70.6)
		"<table>",                 // nested include table
		"badge-ok",                // badge partial via nested include
		"Bob &lt;x&gt;",           // escaped user name
		`<div class="raw"><i>hi</i></div>`,        // raw note
		"&lt;i&gt;hi&lt;/i&gt;",                    // escaped note
		"Use <%= expr %> to output a value.",      // literal delimiters
		"EJS4GO &mdash; 2026",                     // footer include
	)
	mustNotContain(t, body, "summary computed with real JavaScript") // comment dropped
}

// TestEmptyBodyDefaults ensures endpoints supply defaults when no JSON sent.
func TestEmptyBodyDefaults(t *testing.T) {
	_, body := do(t, http.MethodPost, "/render/escaped", "")
	mustContain(t, body, "hello &amp; welcome")
}
