package ospath_test

import (
	"runtime"
	"testing"

	"github.com/vukyn/petkit/internal/ospath"
)

// unix is any platform that is not Windows. These tests name one explicitly
// rather than using runtime.GOOS, so that the Unix answer is asserted on a
// Windows machine too and the Windows answer on this one.
const unix = "linux"

// The defect this package exists for: on Windows `C:\x` and `c:\x` are one
// file, and comparing them byte for byte reports a link that is fine as stale.
func TestWindowsComparesPathsWithoutRegardToCase(t *testing.T) {
	if !ospath.Equal(`C:\Users\me\.claude`, `c:\users\me\.claude`, ospath.Windows) {
		t.Error("two spellings of one Windows path were not recognised as the same place")
	}
}

// The sibling: everywhere else the same two paths are two files, and folding
// case there would make petkit call a genuinely stale link linked.
func TestEverywhereElseCaseIsPartOfTheName(t *testing.T) {
	if ospath.Equal("/home/me/.claude", "/home/me/.CLAUDE", unix) {
		t.Error("two different Unix paths were treated as one place")
	}
	if !ospath.Equal("/home/me/.claude", "/home/me/.claude", unix) {
		t.Error("a path was not equal to itself")
	}
}

// Windows accepts either separator from its callers, so two spellings of one
// path have to compare equal there — and must not on Unix, where a backslash is
// an ordinary character in a file name.
func TestOnlyWindowsTreatsBothSeparatorsAsOne(t *testing.T) {
	if !ospath.Equal(`C:\a\b`, "C:/a/b", ospath.Windows) {
		t.Error("Windows did not accept the forward-slash spelling of a path")
	}
	if ospath.Equal(`/a\b`, "/a/b", unix) {
		t.Error("a Unix file name containing a backslash was treated as two components")
	}
}

// A manifest is written with "/" on every machine; FromSlash is where that stops
// being text and becomes a path.
func TestFromSlashConvertsForWindowsOnly(t *testing.T) {
	if got := ospath.FromSlash("skills/writing-todo", ospath.Windows); got != `skills\writing-todo` {
		t.Errorf("FromSlash for Windows = %q", got)
	}
	if got := ospath.FromSlash("skills/writing-todo", unix); got != "skills/writing-todo" {
		t.Errorf("FromSlash off Windows changed the path to %q", got)
	}
}

// ToSlash is the printing half: a path shown back to the reader is spelled the
// way the manifest spells it.
func TestToSlashConvertsForWindowsOnly(t *testing.T) {
	if got := ospath.ToSlash(`skills\writing-todo`, ospath.Windows); got != "skills/writing-todo" {
		t.Errorf("ToSlash for Windows = %q", got)
	}
	if got := ospath.ToSlash(`a\b`, unix); got != `a\b` {
		t.Errorf("ToSlash off Windows rewrote a legal file name to %q", got)
	}
}

// Under is what the ~ collapse and the doctor survey are built on, so it has to
// answer for a root directory as well as for an ordinary one — a root already
// ends in a separator and a naive prefix would ask for two.
func TestUnderFindsTheRemainder(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		dir   string
		goos  string
		rest  string
		under bool
	}{
		{"ordinary unix directory", "/home/me/.claude/skills/x", "/home/me", unix, ".claude/skills/x", true},
		{"the directory itself", "/home/me", "/home/me", unix, "", true},
		{"a unix root", "/x", "/", unix, "x", true},
		{"not under it at all", "/elsewhere/x", "/home/me", unix, "", false},
		{"a sibling with a shared prefix", "/home/meeting/x", "/home/me", unix, "", false},
		{"windows, differing case", `C:\Users\ME\.claude\skills\x`, `c:\users\me`, ospath.Windows, `.claude\skills\x`, true},
		{"a windows drive root", `C:\x`, `C:\`, ospath.Windows, "x", true},
		{"unix does not fold case", "/home/ME/x", "/home/me", unix, "", false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rest, under := ospath.Under(testCase.path, testCase.dir, testCase.goos)
			if under != testCase.under || rest != testCase.rest {
				t.Errorf("Under(%q, %q, %q) = (%q, %v), want (%q, %v)",
					testCase.path, testCase.dir, testCase.goos, rest, under, testCase.rest, testCase.under)
			}
		})
	}
}

// Key is what a map keyed by "which file is this" uses, so it has to agree with
// Equal: two paths that name one place must not land in two buckets.
func TestKeyAgreesWithEqual(t *testing.T) {
	if ospath.Key(`C:\Users\Me\x`, ospath.Windows) != ospath.Key(`c:\users\me/x`, ospath.Windows) {
		t.Error("two spellings of one Windows path produced two map keys")
	}
	if ospath.Key("/home/me/x", unix) == ospath.Key("/home/ME/x", unix) {
		t.Error("two different Unix paths produced one map key")
	}
}

// Current is the only read of the real platform in the repository; everything
// else is handed the answer.
func TestCurrentIsTheRealPlatform(t *testing.T) {
	if ospath.Current() != runtime.GOOS {
		t.Errorf("Current() = %q, want %q", ospath.Current(), runtime.GOOS)
	}
}

// MFST-006. The defect: validateSource asked path/filepath, which is compiled
// for the host, so `/x` was absolute on macOS and not on Windows and the same
// manifest was refused on one machine and accepted on the other.
//
// ⚠️ Both answers are asserted here, on whichever machine runs the test. That
// is the point: neither spelling may depend on who is asking.
func TestIsAbsAnswersForThePlatformItIsGiven(t *testing.T) {
	cases := []struct {
		path    string
		windows bool
		unix    bool
	}{
		{path: "skills/writing-todo"},
		{path: "./skills/x"},
		{path: "../outside/x"},
		{path: "/etc/skills/x", unix: true},
		{path: "/", unix: true},
		{path: `C:\skills\x`, windows: true},
		{path: "C:/skills/x", windows: true},
		{path: "c:/skills/x", windows: true},
		{path: `\\server\share\x`, windows: true},
		{path: `\skills\x`},
		{path: "C:skills/x"},
	}
	for _, c := range cases {
		if got := ospath.IsAbs(c.path, ospath.Windows); got != c.windows {
			t.Errorf("IsAbs(%q, windows) = %v, want %v", c.path, got, c.windows)
		}
		if got := ospath.IsAbs(c.path, unix); got != c.unix {
			t.Errorf("IsAbs(%q, %s) = %v, want %v", c.path, unix, got, c.unix)
		}
	}
}

// The positive case the refusal owes: a path that is absolute nowhere is
// absolute nowhere, whichever platform is asked.
func TestARelativePathIsAbsoluteOnNoPlatform(t *testing.T) {
	for _, path := range []string{"skills/x", "skills", "x/y/z", ".", ""} {
		if ospath.IsAbs(path, ospath.Windows) || ospath.IsAbs(path, unix) {
			t.Errorf("%q was called absolute", path)
		}
	}
}

// IsAbsAnywhere is the question a rule that travels has to ask, and it is a
// question about the manifest rather than about this machine: a path absolute
// on *either* platform is refused on *both*, so the same petkit.yaml is read
// the same way everywhere.
func TestIsAbsAnywhereAsksBothPlatformsAtOnce(t *testing.T) {
	for _, path := range []string{"/etc/skills/x", "C:/skills/x", `C:\skills\x`, `\\server\share\x`} {
		if !ospath.IsAbsAnywhere(path) {
			t.Errorf("%q is absolute on one platform and IsAbsAnywhere said no", path)
		}
	}
	// The positive case: a manifest's own spelling stays acceptable.
	for _, path := range []string{"skills/writing-todo", "skills/nested/../x", ""} {
		if ospath.IsAbsAnywhere(path) {
			t.Errorf("%q is relative everywhere and IsAbsAnywhere said yes", path)
		}
	}
}

// CLI-012. ⚠️ filepath.Clean is compiled for the host, so a Layout carrying
// GOOS="linux" still cleaned its paths the Windows way when the suite ran on
// Windows: the injected platform and path/filepath disagreed, and filepath won.
// Four tests failed for that reason alone and none of them was a defect in the
// product.
func TestCleanAnswersForThePlatformItIsGiven(t *testing.T) {
	cases := []struct {
		path    string
		windows string
		unix    string
	}{
		{path: "", windows: ".", unix: "."},
		{path: ".", windows: ".", unix: "."},
		{path: "a/b/../c", windows: `a\c`, unix: "a/c"},
		{path: "a//b", windows: `a\b`, unix: "a/b"},
		{path: "a/b/", windows: `a\b`, unix: "a/b"},
		{path: "./a/./b", windows: `a\b`, unix: "a/b"},
		{path: "../x", windows: `..\x`, unix: "../x"},
		{path: "/home/me/.claude", windows: `\home\me\.claude`, unix: "/home/me/.claude"},
		// ⚠️ Off Windows a backslash is an ordinary character in a file name, so
		// this is one segment ending in one, and nothing about it is a trailing
		// separator to strip. Both answers are right; they are answers to
		// different questions.
		{path: `C:\Users\me\`, windows: `C:\Users\me`, unix: `C:\Users\me\`},
		{path: "C:/Users/me/x", windows: `C:\Users\me\x`, unix: "C:/Users/me/x"},
	}
	for _, c := range cases {
		if got := ospath.Clean(c.path, ospath.Windows); got != c.windows {
			t.Errorf("Clean(%q, windows) = %q, want %q", c.path, got, c.windows)
		}
		if got := ospath.Clean(c.path, unix); got != c.unix {
			t.Errorf("Clean(%q, %s) = %q, want %q", c.path, unix, got, c.unix)
		}
	}
}

// Join is Clean's companion: the manifest writes "/" on every machine, and the
// separator arrives when a target stops being text and becomes a path.
func TestJoinUsesTheSeparatorOfThePlatformItIsGiven(t *testing.T) {
	cases := []struct {
		elements []string
		windows  string
		unix     string
	}{
		{elements: []string{`C:\Users\me`, "skills/x"}, windows: `C:\Users\me\skills\x`, unix: `C:\Users\me/skills/x`},
		{elements: []string{"/home/me", ".claude", "skills"}, windows: `\home\me\.claude\skills`, unix: "/home/me/.claude/skills"},
		{elements: []string{"a", "", "b"}, windows: `a\b`, unix: "a/b"},
		{elements: []string{"a", "../b"}, windows: "b", unix: "b"},
		{elements: nil, windows: "", unix: ""},
	}
	for _, c := range cases {
		if got := ospath.Join(ospath.Windows, c.elements...); got != c.windows {
			t.Errorf("Join(windows, %q) = %q, want %q", c.elements, got, c.windows)
		}
		if got := ospath.Join(unix, c.elements...); got != c.unix {
			t.Errorf("Join(%s, %q) = %q, want %q", unix, c.elements, got, c.unix)
		}
	}
}
