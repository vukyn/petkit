// Package ospath holds the handful of path questions whose answer depends on
// which operating system is being talked about: whether two paths that differ
// in case name the same file, and which character separates the components.
//
// ⚠️ Every function here takes the platform as a **parameter** rather than
// reading runtime.GOOS, and that is the whole reason the package exists.
// path/filepath is compiled for the host: on macOS filepath.FromSlash can only
// ever be the identity and filepath.Clean can only ever produce a Unix path, so
// a Windows branch written on top of them cannot be watched from the machine
// this repository is developed on. With the platform as an argument a macOS test
// runs the Windows answer and the Unix one side by side, in the same process,
// with no build tag and no second machine. Current() is the only place the real
// platform is read, and only main is meant to call it.
package ospath

import (
	"runtime"
	"strings"
)

// Windows is the GOOS value whose filesystem folds case and whose separator is
// a backslash. It is the only platform this package treats specially: the rest
// of the world agrees with Unix closely enough for petkit's purposes.
//
// ⚠️ macOS is deliberately NOT in here even though its default filesystem also
// folds case. A case-folding comparison on macOS would be a guess about how the
// volume was formatted (APFS can be either), and getting it wrong the generous
// way would make petkit call a genuinely stale link "linked".
const Windows = "windows"

// Current is the platform this binary was built for. It is the only read of
// runtime.GOOS in the repository; everything else is handed the answer.
func Current() string { return runtime.GOOS }

// FoldsCase reports whether goos compares path names without regard to case.
func FoldsCase(goos string) bool { return goos == Windows }

// Separator is the character goos puts between path components.
func Separator(goos string) string {
	if goos == Windows {
		return `\`
	}
	return "/"
}

// FromSlash is filepath.FromSlash with the platform as a parameter: it turns
// the forward slashes a manifest is written with into the separator goos uses.
//
// A manifest says `skills/writing-todo` on every machine — that is the point of
// a manifest — so the conversion happens wherever one of those strings stops
// being text and becomes a path.
func FromSlash(path, goos string) string {
	if goos != Windows {
		return path
	}
	return strings.ReplaceAll(path, "/", `\`)
}

// ToSlash is FromSlash's inverse, used when a path is printed: a path shown back
// to the reader is spelled the way the manifest spells it.
func ToSlash(path, goos string) string {
	if goos != Windows {
		return path
	}
	return strings.ReplaceAll(path, `\`, "/")
}

// Equal reports whether two paths name the same place on goos.
//
// On Windows that means `C:\x` and `c:\x` are one path, and so are `C:\a\b` and
// `C:\a/b` — the API accepts either separator. Everywhere else it is a byte
// comparison, because everywhere else a backslash is an ordinary character in a
// file name and folding case would be a guess.
//
// ⚠️ The comparison is strings.EqualFold, which is Unicode simple folding, not
// the uppercase table Windows itself uses. They agree on ASCII — drive letters,
// and every name petkit puts in a manifest — and disagree only on exotic
// characters, where the cost is a `stale` reported for a link that is fine.
// That is the direction to be wrong in: it makes petkit rewrite a link, never
// leave a wrong one in place.
func Equal(a, b, goos string) bool {
	if !FoldsCase(goos) {
		return a == b
	}
	return strings.EqualFold(normalise(a, goos), normalise(b, goos))
}

// Key is the form of a path to use as a map key when the map is keyed by
// "which file is this". Two paths that are Equal produce the same Key.
func Key(path, goos string) string {
	if !FoldsCase(goos) {
		return path
	}
	return strings.ToLower(normalise(path, goos))
}

// Under reports whether path is dir or lies inside it, and returns what is left
// of path below dir. The remainder keeps the separators it arrived with.
//
// dir is allowed to be a root (`/`, or `C:\`): the trailing separator is
// trimmed before the prefix is rebuilt, so the prefix stays exactly one
// separator long either way.
func Under(path, dir, goos string) (string, bool) {
	if Equal(path, dir, goos) {
		return "", true
	}
	trimmed := strings.TrimRight(dir, separators(goos))
	for _, separator := range strings.Split(separators(goos), "") {
		prefix := trimmed + separator
		if len(path) > len(prefix) && Equal(path[:len(prefix)], prefix, goos) {
			return path[len(prefix):], true
		}
	}
	return "", false
}

// normalise makes the two spellings of a Windows separator one spelling, so a
// comparison does not have to know which the caller used.
func normalise(path, goos string) string {
	if goos != Windows {
		return path
	}
	return strings.ReplaceAll(path, "/", `\`)
}

// separators is every character goos accepts between components.
func separators(goos string) string {
	if goos == Windows {
		return `\/`
	}
	return "/"
}
