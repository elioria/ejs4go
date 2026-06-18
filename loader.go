package ejs4go

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Loader resolves an include() path (relative to the including template) to
// template source text. Implementations back includes with a filesystem, an
// embedded FS, a map, etc.
//
// `from` is the logical filename of the including template (may be empty for
// string templates). `name` is the path passed to include(). The returned
// resolved name is used as the filename of the sub-template (for nested
// includes and error messages).
type Loader interface {
	Load(from, name string) (src string, resolved string, err error)
}

// FileLoader loads includes from the filesystem. Includes are resolved
// relative to the including template's directory, falling back to Root.
// A ".ejs" extension is appended when the resolved path has no extension.
type FileLoader struct {
	// Root is the base directory for includes whose including template has
	// no known directory (e.g. string templates). Defaults to ".".
	Root string
	// Ext is appended to extension-less include names. Defaults to ".ejs".
	Ext string
}

// NewFileLoader returns a FileLoader rooted at dir.
func NewFileLoader(dir string) *FileLoader {
	return &FileLoader{Root: dir, Ext: ".ejs"}
}

// Load implements Loader.
func (l *FileLoader) Load(from, name string) (string, string, error) {
	root := l.Root
	if root == "" {
		root = "."
	}
	ext := l.Ext
	if ext == "" {
		ext = ".ejs"
	}

	nameOS := filepath.FromSlash(name)

	// An absolute include name (or top-level FromFile path) is used verbatim;
	// joining it onto base/root would duplicate the prefix.
	var resolved string
	if filepath.IsAbs(nameOS) {
		resolved = nameOS
	} else {
		base := root
		if from != "" {
			base = filepath.Dir(from)
		}
		resolved = filepath.Join(base, nameOS)
	}
	if filepath.Ext(resolved) == "" {
		resolved += ext
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		// Retry against Root if the relative resolution missed.
		alt := filepath.Join(root, filepath.FromSlash(name))
		if filepath.Ext(alt) == "" {
			alt += ext
		}
		data2, err2 := os.ReadFile(alt)
		if err2 != nil {
			return "", "", err
		}
		return string(data2), alt, nil
	}
	return string(data), resolved, nil
}

// MapLoader serves includes from an in-memory map of name -> source. Keys are
// matched both verbatim and with a ".ejs" suffix, and paths are normalized
// with forward slashes.
type MapLoader struct {
	Sources map[string]string
}

// NewMapLoader returns a MapLoader over the given sources.
func NewMapLoader(sources map[string]string) *MapLoader {
	return &MapLoader{Sources: sources}
}

// Load implements Loader.
func (l *MapLoader) Load(from, name string) (string, string, error) {
	// Resolve relative to the including template using slash paths.
	var base string
	if from != "" {
		base = path.Dir(filepath.ToSlash(from))
	}
	candidates := []string{
		path.Join(base, name),
		path.Join(base, name) + ".ejs",
		name,
		name + ".ejs",
	}
	for _, c := range candidates {
		c = strings.TrimPrefix(c, "./")
		if src, ok := l.Sources[c]; ok {
			return src, c, nil
		}
	}
	return "", "", &RuntimeError{Msg: "include not found: " + name}
}
