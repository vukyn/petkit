// Package version reports the build's version.
//
// The version is the git tag and nothing else. There is no constant to forget
// to bump and no -ldflags -X to forget to pass: the Go toolchain records the
// module version it built, and this package only formats it.
package version

import (
	"regexp"
	"runtime/debug"
	"strings"
)

// Devel is what a build that carries no module version reports — `go run`, or a
// binary built inside the module without `go install module@version`.
const Devel = "(devel)"

// pseudoVersion matches the versions the module proxy synthesises for an
// untagged commit. There are two shapes, and the separator before the
// timestamp differs between them: v0.0.0-20260914150405-abcdef123456 when no
// tag precedes the commit, and v1.2.3-0.20260914150405-abcdef123456 when one
// does.
//
// ⚠️ A build from a dirty working tree carries a `+dirty` build metadata suffix,
// and an anchored pattern without it matches nothing — so the whole
// pseudo-version used to be printed where a revision was meant. The suffix is
// optional here and kept in the output, because "which commit" and "plus
// uncommitted changes" are two different answers and a reader needs both.
var pseudoVersion = regexp.MustCompile(`^v.*[-.][0-9]{14}-[0-9a-f]{12}(\+[0-9A-Za-z.-]+)?$`)

// Current reads the version out of the build info.
func Current() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Devel
	}
	return Format(info.Main.Version)
}

// ModulePath reads the module this binary was built from, out of the same build
// info the version comes from. It is empty when the build carries no module
// path at all.
//
// ⚠️ This is what `petkit setup` clones, and it is deliberately not a constant.
// A fork installed from its own module path clones itself; a constant would send
// it to somebody else's repository, and the mistake would look like success.
func ModulePath() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return strings.TrimSpace(info.Main.Path)
}

// Format leaves a tag whole and trims a pseudo-version to its revision, which
// is the only part of a pseudo-version anybody can act on.
func Format(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Devel
	}
	if pseudoVersion.MatchString(raw) {
		return raw[strings.LastIndex(raw, "-")+1:]
	}
	return raw
}
