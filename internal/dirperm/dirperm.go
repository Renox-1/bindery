// Package dirperm resolves the directory mode used for library content
// directories created by the importer (author, book and audiobook folders).
//
// The default remains the historical 0750. Operators whose storage rewrites
// client IDs server-side (NFS exports with all_squash/mapall, root-squash
// setups) or who need other readers to traverse these directories can relax
// the restriction with BINDERY_DIR_MODE.
package dirperm

import (
	"os"
	"strings"
)

// DefaultLibraryMode is the historical mode for importer-created library
// directories.
const DefaultLibraryMode os.FileMode = 0o750

// allowedModes is the closed set of modes BINDERY_DIR_MODE accepts, keyed
// without the leading zero; both spellings are accepted at lookup. A closed
// whitelist (rather than free-form octal parsing) keeps operator mistakes —
// "0o755", a stray space, a decimal value — from locking the importer out of
// its own directories, and makes the accepted values easy to document.
var allowedModes = map[string]os.FileMode{
	"700": 0o700,
	"750": 0o750,
	"755": 0o755,
	"770": 0o770,
	"775": 0o775,
	"777": 0o777,
}

// Library returns the mode to use when creating library content directories.
//
// It reads BINDERY_DIR_MODE on every call. The value must be one of the
// whitelisted modes, written with or without a leading zero (e.g. "0755" or
// "755"). Any unset, empty or unrecognised value falls back to 0o750, the
// historical default; config.Validate warns at startup when the value is set
// but unrecognised.
func Library() os.FileMode {
	return parse(os.Getenv("BINDERY_DIR_MODE"))
}

// Valid reports whether raw is a BINDERY_DIR_MODE value the package accepts.
// Empty or whitespace-only values are valid: they mean "use the default".
func Valid(raw string) bool {
	n := normalize(raw)
	if n == "" {
		return true
	}
	_, ok := allowedModes[n]
	return ok
}

func normalize(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "0")
}

func parse(raw string) os.FileMode {
	if m, ok := allowedModes[normalize(raw)]; ok {
		return m
	}
	return DefaultLibraryMode
}
