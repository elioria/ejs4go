package main

// This file holds every EJS template the API renders. Templates are grouped
// by the capability they exercise so the integration tests can assert each in
// isolation, plus one large "dashboard" template that combines all features.

// ---------------------------------------------------------------------------
// Partials served through the MapLoader for include() coverage.
// ---------------------------------------------------------------------------

var partials = map[string]string{
	// Simple partial used by the include endpoint.
	"partials/badge.ejs": `<span class="badge badge-<%= kind %>"><%= label %></span>`,

	// Partial that itself includes another partial -> nested include coverage.
	"partials/userRow.ejs": `<tr>
  <td><%= user.name %></td>
  <td><%- include('partials/badge', { kind: user.active ? 'ok' : 'off', label: user.active ? 'active' : 'inactive' }) %></td>
  <td><%= user.score %></td>
</tr>`,

	// Partial that renders a list, exercising includes inside a loop.
	"partials/userTable.ejs": `<table>
  <thead><tr><th>Name</th><th>Status</th><th>Score</th></tr></thead>
  <tbody>
<% users.forEach(function(user) { -%>
<%- include('partials/userRow', { user: user }) %>
<% }) -%>
  </tbody>
</table>`,

	// Header/footer partials for the big dashboard.
	"partials/header.ejs": `<header><h1><%= title %></h1><small>generated for <%= owner %></small></header>`,
	"partials/footer.ejs": `<footer>EJS4GO &mdash; <%= year %></footer>`,
}

// ---------------------------------------------------------------------------
// Per-capability templates.
// ---------------------------------------------------------------------------

const (
	// tmplText: plain text passthrough, no tags.
	tmplText = `Welcome to the EJS4GO demo server.`

	// tmplEscaped: <%= %> HTML-escapes output.
	tmplEscaped = `<p><%= message %></p>`

	// tmplRaw: <%- %> emits unescaped HTML.
	tmplRaw = `<div><%- html %></div>`

	// tmplEval: <% %> control flow with no direct output.
	tmplEval = `<ul>
<% for (var i = 1; i <= n; i++) { -%>
  <li>Item <%= i %></li>
<% } -%>
</ul>`

	// tmplComment: <%# %> produces nothing.
	tmplComment = `start<%# this secret note is never rendered %>end`

	// tmplLiteral: <%% / %%> emit literal delimiters.
	tmplLiteral = `To print a value write <%%= value %%> in your template.`

	// tmplTrim: -%> trims the trailing newline so list items stay tight.
	tmplTrim = `<% items.forEach(function(it) { -%>
<%= it %>
<% }) -%>
DONE`

	// tmplSlurp: <%_ and _%> slurp surrounding whitespace.
	tmplSlurp = `LEFT    <%_ var k = 1 _%>    RIGHT`

	// tmplRmWhitespace: rmWhitespace option strips line padding.
	tmplRmWhitespace = `   <% var a = 1 %>
   <%= a %>   `

	// tmplCustomDelim: custom [%  %] delimiters.
	tmplCustomDelim = `Hello [%= name %], you have [%= count %] messages.`

	// tmplStrict: strict mode disables `with`, locals via the locals object.
	tmplStrict = `<%= locals.greeting %>, <%= locals.who %>!`

	// tmplInclude: single include with data.
	tmplInclude = `Status: <%- include('partials/badge', { kind: 'ok', label: status }) %>`

	// tmplNestedInclude: include a partial that includes another.
	tmplNestedInclude = `<%- include('partials/userTable', { users: users }) %>`

	// tmplRealJS exercises a broad slice of the JS language and stdlib that
	// goja supports: array methods, arrow functions, template literals,
	// destructuring-free map/filter/reduce, JSON, Math, ternary, String methods.
	tmplRealJS = `<%
  var nums = data.numbers;
  var doubled = nums.map(function(x){ return x * 2; });
  var evens = nums.filter(function(x){ return x % 2 === 0; });
  var total = nums.reduce(function(a, b){ return a + b; }, 0);
  var labels = nums.map(function(x){ return 'n' + x; }).join(',');
-%>
doubled=<%= doubled.join('-') %>
evens=<%= evens.join('-') %>
total=<%= total %>
max=<%= Math.max.apply(null, nums) %>
labels=<%= labels %>
upper=<%= data.word.toUpperCase() %>
json=<%- JSON.stringify({ count: nums.length, total: total }) %>
parity=<%= total % 2 === 0 ? 'even' : 'odd' %>`

	// tmplBadSyntax: intentionally unterminated, drives the syntax-error path.
	tmplBadSyntax = `Oops <%= unterminated`

	// tmplBadRuntime: references an undeclared identifier at render time.
	tmplBadRuntime = `<%= doesNotExist.value %>`
)

// tmplDashboard is the large, complex template that combines every capability:
// header/footer includes, escaped + raw output, loops, conditionals, nested
// includes inside a loop, real JS computation, comments, and literal delimiters.
const tmplDashboard = `<!doctype html>
<html>
<body>
<%- include('partials/header', { title: title, owner: owner }) %>

<%# ---- summary computed with real JavaScript ---- %>
<%
  var scores = users.map(function(u){ return u.score; });
  var avg = scores.reduce(function(a, b){ return a + b; }, 0) / (scores.length || 1);
  var top = users.slice().sort(function(a, b){ return b.score - a.score; })[0];
  var activeCount = users.filter(function(u){ return u.active; }).length;
-%>
<section class="summary">
  <p>Users: <%= users.length %> (active: <%= activeCount %>)</p>
  <p>Average score: <%= avg.toFixed(1) %></p>
  <p>Top user: <%= top ? top.name : 'n/a' %></p>
  <p>Mood: <%= avg >= 50 ? 'healthy' : 'needs attention' %></p>
</section>

<%# ---- table built via nested includes inside the partial ---- %>
<section class="table">
<%- include('partials/userTable', { users: users }) %>
</section>

<%# ---- raw vs escaped demonstration ---- %>
<section class="notes">
  <div class="raw"><%- note %></div>
  <div class="escaped"><%= note %></div>
</section>

<%# ---- literal delimiter help text ---- %>
<p class="help">Use <%%= expr %%> to output a value.</p>

<%- include('partials/footer', { year: year }) %>
</body>
</html>`
