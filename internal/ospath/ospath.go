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
	// ⚠️ path, not path/filepath. This package's whole job is to answer for a
	// platform it is told about rather than the one it was compiled for, and
	// path/filepath cannot do that. path is separator-agnostic arithmetic on
	// "/", which is what the Windows answers here are built out of.
	gopath "path"
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

// IsAbs reports whether path is absolute on goos.
//
// ⚠️ This is path/filepath's question with the platform as a parameter, and
// MFST-006 is why it had to become one. filepath.IsAbs is compiled for the
// host, and the two platforms disagree in **both** directions: `/x` is absolute
// on Unix and not on Windows, `C:/x` is absolute on Windows and not on Unix. A
// manifest rule asked of the host is a rule that answers differently per
// machine, which is the one thing petkit.yaml may not do.
//
// Windows counts two shapes as absolute, and `\x` is deliberately not one of
// them: it is rooted on the current drive rather than absolute, and filepath
// agrees.
func IsAbs(path, goos string) bool {
	if goos != Windows {
		return strings.HasPrefix(path, "/")
	}
	separator := func(index int) bool {
		return index < len(path) && strings.IndexByte(separators(goos), path[index]) >= 0
	}
	switch {
	case separator(0) && separator(1): // \\server\share
		return true
	case len(path) >= 3 && path[1] == ':' && separator(2): // C:\x
		return isDriveLetter(path[0])
	}
	return false
}

// IsAbsAnywhere reports whether path is absolute on any platform petkit runs
// on, and it is the form a rule about a **manifest** wants rather than IsAbs.
//
// ⚠️ A manifest travels: the same petkit.yaml is read on a Mac and on a Windows
// machine, so a path absolute on either has to be refused on both. Asking only
// the host is how MFST-006 happened — and MFST-005, the backslash refusal, is
// the same argument reached a day earlier from the other end.
func IsAbsAnywhere(path string) bool {
	return IsAbs(path, Windows) || IsAbs(path, anyUnix)
}

// anyUnix is a platform that is not Windows. Which one does not matter: this
// package treats every non-Windows platform alike, and naming one keeps
// IsAbsAnywhere from having to read runtime.GOOS.
const anyUnix = "linux"

// Clean is filepath.Clean with the platform as a parameter: it removes `.`,
// resolves `..` lexically, collapses repeated separators, and spells the result
// with the separator goos uses.
//
// ⚠️ CLI-012 is why this exists. filepath.Clean is compiled for the host, so a
// Layout carrying GOOS="linux" still cleaned the Windows way when the suite ran
// on Windows: the injected platform and path/filepath disagreed, and filepath
// won. Four tests failed for that reason and not one was a defect in the product.
//
// ⚠️ A UNC path is not preserved: `\\server\share` cleans to `\server\share`,
// because the leading double separator is collapsed like any other. petkit never
// builds one — a manifest may not contain a backslash at all (`MFST-005`) and a
// target must start with `~/` — so the case is documented rather than handled.
func Clean(path, goos string) string {
	if goos != Windows {
		return gopath.Clean(path)
	}
	return FromSlash(gopath.Clean(ToSlash(path, goos)), goos)
}

// Join is filepath.Join with the platform as a parameter: the elements are
// joined with the separator goos uses and the result is Cleaned.
//
// The manifest writes "/" on every machine — that is what makes it a manifest —
// so the separator arrives here, where a target stops being text and becomes a
// path the filesystem will be asked about.
func Join(goos string, elements ...string) string {
	present := make([]string, 0, len(elements))
	for _, element := range elements {
		if element != "" {
			present = append(present, element)
		}
	}
	if len(present) == 0 {
		return ""
	}
	return Clean(strings.Join(present, "/"), goos)
}

// isDriveLetter reports whether b can name a Windows drive.
func isDriveLetter(b byte) bool {
	return ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}
