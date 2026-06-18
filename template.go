package ejs4go

import (
	"fmt"

	"github.com/dop251/goja"
)

// Template is a parsed and compiled EJS template ready to be executed.
//
// A Template is safe to reuse and is independent of any goja runtime: the
// generated JavaScript source is compiled to a goja *Program once and a fresh
// runtime is created per Execute call (goja runtimes are single-goroutine).
type Template struct {
	opts    Options
	source  string       // generated JS function-expression source
	program *goja.Program // compiled program

	// name is the logical filename, used for includes and errors.
	name string
}

// FromString parses and compiles an EJS template from source text.
func FromString(src string, opts ...Option) (*Template, error) {
	o := DefaultOptions()
	for _, fn := range opts {
		fn(&o)
	}
	o.normalize()
	return fromStringWith(src, o)
}

// fromStringWith builds a Template from an already-resolved Options value.
func fromStringWith(src string, o Options) (*Template, error) {
	segs, err := parse(src, &o)
	if err != nil {
		return nil, err
	}
	body := compile(segs, &o)
	fnSrc := wrapFunction(body, o.LocalsName)

	prog, err := goja.Compile(o.Filename, fnSrc, o.Strict)
	if err != nil {
		return nil, &CompileError{Msg: err.Error(), File: o.Filename}
	}

	return &Template{
		opts:    o,
		source:  fnSrc,
		program: prog,
		name:    o.Filename,
	}, nil
}

// FromFile reads, parses, and compiles an EJS template from a file. The
// filename is recorded for relative include resolution; if no Loader was
// provided, a FileLoader rooted at the file's directory is installed.
func FromFile(filename string, opts ...Option) (*Template, error) {
	o := DefaultOptions()
	for _, fn := range opts {
		fn(&o)
	}
	o.Filename = filename
	if o.Loader == nil {
		o.Loader = NewFileLoader(".")
	}
	o.normalize()

	loader := o.Loader
	src, resolved, err := loader.Load("", filename)
	if err != nil {
		return nil, err
	}
	o.Filename = resolved
	return fromStringWith(src, o)
}

// Source returns the generated JavaScript function-expression source. Useful
// for debugging the compilation step.
func (t *Template) Source() string { return t.source }

// Execute renders the template with the given locals and returns the output.
//
// A new goja runtime is created for the call. The escape function and the
// include() helper are installed as globals; the compiled function is then
// invoked with the locals object.
func (t *Template) Execute(locals map[string]any) (string, error) {
	vm := goja.New()
	if err := t.install(vm); err != nil {
		return "", err
	}
	return t.render(vm, locals)
}

// install wires the runtime-wide helpers (escapeFn, include) into vm. These
// are shared by the top-level template and every include it triggers.
func (t *Template) install(vm *goja.Runtime) error {
	escapeFn := t.opts.EscapeFunc
	if escapeFn == nil {
		escapeFn = HTMLEscape
	}
	if err := vm.Set("escapeFn", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if goja.IsUndefined(arg) || goja.IsNull(arg) {
			return vm.ToValue("")
		}
		return vm.ToValue(escapeFn(arg.String()))
	}); err != nil {
		return err
	}

	// include(name, data) compiles and runs a sub-template against the same
	// runtime, returning its rendered output as a string. Errors are thrown
	// as JS exceptions so they surface at the include site.
	includeFn := func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		var data map[string]any
		if d := call.Argument(1); !goja.IsUndefined(d) && !goja.IsNull(d) {
			if exported, ok := d.Export().(map[string]any); ok {
				data = exported
			} else {
				// Coerce arbitrary objects via Export then best-effort map.
				data = toStringMap(d.Export())
			}
		}
		out, err := t.renderInclude(vm, name, data)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return vm.ToValue(out)
	}
	if err := vm.Set("include", includeFn); err != nil {
		return err
	}

	// __rethrow is referenced by Debug-mode try/catch wrappers.
	if err := vm.Set("__rethrow", func(call goja.FunctionCall) goja.Value {
		panic(call.Argument(0))
	}); err != nil {
		return err
	}
	return nil
}

// render compiles-and-runs this template's program in vm with locals.
func (t *Template) render(vm *goja.Runtime, locals map[string]any) (string, error) {
	val, err := vm.RunProgram(t.program)
	if err != nil {
		return "", t.wrapRuntimeErr(err)
	}
	fn, ok := goja.AssertFunction(val)
	if !ok {
		return "", &RuntimeError{Msg: "compiled template is not a function", File: t.name}
	}
	out, err := fn(goja.Undefined(), vm.ToValue(locals))
	if err != nil {
		return "", t.wrapRuntimeErr(err)
	}
	return out.String(), nil
}

// renderInclude resolves name via the loader, compiles a sub-template, and
// renders it against the shared runtime vm.
func (t *Template) renderInclude(vm *goja.Runtime, name string, data map[string]any) (string, error) {
	loader := t.opts.Loader
	if loader == nil {
		return "", &RuntimeError{Msg: "include() used but no Loader configured", File: t.name}
	}
	src, resolved, err := loader.Load(t.name, name)
	if err != nil {
		return "", &RuntimeError{Msg: fmt.Sprintf("cannot load include %q: %v", name, err), File: t.name}
	}

	subOpts := t.opts
	subOpts.Filename = resolved
	sub, err := fromStringWith(src, subOpts)
	if err != nil {
		return "", err
	}
	// Reuse the same runtime so escapeFn/include remain available.
	return sub.render(vm, data)
}

// wrapRuntimeErr converts a goja error into a RuntimeError carrying the JS
// exception value's message when available.
func (t *Template) wrapRuntimeErr(err error) error {
	if ex, ok := err.(*goja.Exception); ok {
		return &RuntimeError{Msg: ex.Value().String(), File: t.name}
	}
	return &RuntimeError{Msg: err.Error(), File: t.name}
}

// toStringMap best-effort converts an exported JS value into map[string]any.
func toStringMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
