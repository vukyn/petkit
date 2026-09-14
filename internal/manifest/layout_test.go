package manifest_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
)

// windowsMachine is a Windows home directory described to code running on this
// one. Config is spelled out rather than derived, because deriving it would go
// through the host's filepath.Join and produce a Unix answer here.
func windowsMachine() manifest.Layout {
	return manifest.Layout{
		Home:   `C:\Users\me`,
		Config: `C:\Users\me\.claude`,
		GOOS:   ospath.Windows,
	}
}

// A manifest target is written with "/" on every machine. Under a Windows root
// the separators in it have to become backslashes, or the path names a file
// called `skills/writing-todo` instead of a file inside a directory.
//
// ⚠️ The expectation joins the root to a remainder spelled with a BACKSLASH, so
// it is right on both hosts: this machine's filepath.Join contributes its own
// separator between the two and Windows' contributes its own, and either way the
// remainder that came out of the manifest has been converted. Dropping the
// conversion leaves a "/" in the remainder and the comparison fails here.
func TestAManifestTargetResolvesUnderAWindowsRoot(t *testing.T) {
	layout := windowsMachine()

	got := layout.Resolve("~/.claude/skills/writing-todo")
	want := filepath.Join(`C:\Users\me\.claude`, `skills\writing-todo`)
	if got != want {
		t.Errorf("Resolve under a Windows root = %q, want %q", got, want)
	}
}

// The sibling: on this machine the same target resolves the way it always has,
// which is what makes the change a portability fix rather than a rewrite.
func TestTheSameTargetResolvesNativelyOnThisMachine(t *testing.T) {
	home := t.TempDir()
	layout := manifest.NewLayout(home, ospath.Current(), nil)

	got := layout.Resolve("~/.claude/skills/writing-todo")
	want := filepath.Join(home, ".claude", "skills", "writing-todo")
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}

// A source is written with "/" too, and crosses the same boundary.
func TestASourceResolvesUnderAWindowsRoot(t *testing.T) {
	item := manifest.Item{ID: "skill/writing-todo", Source: "skills/writing-todo"}

	got := item.SourcePath(`C:\Users\me\.petkit`, ospath.Windows)
	want := filepath.Join(`C:\Users\me\.petkit`, `skills\writing-todo`)
	if got != want {
		t.Errorf("SourcePath under a Windows root = %q, want %q", got, want)
	}
}

// A path printed back is spelled the way the manifest spells it: with "/",
// on every platform, so the line in `petkit status` and the line in petkit.yaml
// are the same string.
func TestDisplaySpellsAPathTheWayTheManifestDoes(t *testing.T) {
	layout := windowsMachine()

	if got := layout.Display(`C:\Users\me\.claude\skills\writing-todo`); got != "~/.claude/skills/writing-todo" {
		t.Errorf("Display = %q", got)
	}
	// Same file, other case — Windows would not tell them apart and neither
	// may the collapse, or the path prints as an absolute one out of nowhere.
	if got := layout.Display(`c:\users\ME\.claude\skills\writing-todo`); got != "~/.claude/skills/writing-todo" {
		t.Errorf("Display of a differently-cased path = %q", got)
	}
	if got := layout.Display(`C:\Users\me`); got != "~" {
		t.Errorf("Display of the home directory itself = %q", got)
	}
	// Not under the home directory: printed as it is, because writing it as
	// ~/... would name a place it is not.
	if got := layout.Display(`D:\elsewhere\x`); got != `D:\elsewhere\x` {
		t.Errorf("Display of a path outside the home directory = %q", got)
	}
}

// The sibling: off Windows, case is part of the name, so a differently-cased
// path is a different path and must not be collapsed.
func TestDisplayDoesNotFoldCaseOffWindows(t *testing.T) {
	layout := manifest.Layout{Home: "/home/me", Config: "/home/me/.claude", GOOS: "linux"}

	if got := layout.Display("/home/ME/.claude/x"); got != "/home/ME/.claude/x" {
		t.Errorf("a differently-cased Unix path was collapsed to %q", got)
	}
	if got := layout.Display("/home/me/.claude/x"); got != "~/.claude/x" {
		t.Errorf("Display = %q", got)
	}
}

// CLAUDE_CONFIG_DIR is what Claude Code itself honours. petkit hard-coded
// ~/.claude, so on a machine that sets it every link landed in a directory
// nothing reads. ⚠️ This is wrong on every platform, not only on Windows.
func TestClaudeConfigDirDecidesWhereTheTargetsGo(t *testing.T) {
	home := t.TempDir()
	elsewhere := t.TempDir()
	layout := manifest.NewLayout(home, ospath.Current(), func(name string) string {
		if name == manifest.EnvConfigDir {
			return elsewhere
		}
		return ""
	})

	if layout.Config != elsewhere {
		t.Errorf("Config = %q, want %q", layout.Config, elsewhere)
	}
	if got, want := layout.Resolve("~/.claude/skills/x"), filepath.Join(elsewhere, "skills", "x"); got != want {
		t.Errorf("a ~/.claude target resolved to %q, want %q", got, want)
	}
	if got, want := layout.SkillsDir(), filepath.Join(elsewhere, "skills"); got != want {
		t.Errorf("doctor would survey %q, want %q", got, want)
	}
	// ⚠️ Only the configuration directory moves. A target that is not under
	// ~/.claude is still relative to the home directory — the variable names
	// where Claude Code's configuration lives, not where the user does.
	if got, want := layout.Resolve("~/notes/x"), filepath.Join(home, "notes", "x"); got != want {
		t.Errorf("an unrelated target followed the variable: %q, want %q", got, want)
	}
}

// The positive sibling: unset, and set-but-empty, both leave the default alone.
func TestWithoutClaudeConfigDirTheDefaultIsUnderHome(t *testing.T) {
	home := t.TempDir()
	for name, env := range map[string]func(string) string{
		"no environment at all": nil,
		"unset":                 func(string) string { return "" },
		"set to the empty string": func(name string) string {
			if name == manifest.EnvConfigDir {
				return ""
			}
			return ""
		},
	} {
		t.Run(name, func(t *testing.T) {
			layout := manifest.NewLayout(home, ospath.Current(), env)
			if want := filepath.Join(home, ".claude"); layout.Config != want {
				t.Errorf("Config = %q, want %q", layout.Config, want)
			}
		})
	}
}

// `~` is the only templating petkit does, and the variable does not get an
// exemption from that rule.
func TestClaudeConfigDirExpandsATilde(t *testing.T) {
	home := t.TempDir()
	layout := manifest.NewLayout(home, ospath.Current(), func(name string) string {
		if name == manifest.EnvConfigDir {
			return "~/somewhere/claude"
		}
		return ""
	})

	if want := filepath.Join(home, "somewhere", "claude"); layout.Config != want {
		t.Errorf("Config = %q, want %q", layout.Config, want)
	}
}

// A backslash in a manifest path is refused on every platform, because the
// manifest travels: `~/..\elsewhere` is an escape from the home directory on
// Windows and an ordinary file name on macOS, so the machine that wrote the
// manifest would pass a check the machine that reads it fails.
func TestAPathWrittenWithBackslashesIsRefused(t *testing.T) {
	for name, body := range map[string]string{
		"target": `version: 1
items:
  - id: skill/x
    kind: skill
    source: skills/x
    target: ~/.claude\skills\x
`,
		"an escaping target that only Windows would resolve": `version: 1
items:
  - id: skill/x
    kind: skill
    source: skills/x
    target: ~/..\elsewhere\x
`,
		"source": `version: 1
items:
  - id: skill/x
    kind: skill
    source: skills\x
    target: ~/.claude/skills/x
`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := manifest.Parse([]byte(body), t.TempDir())
			if err == nil {
				t.Fatal("a path written with backslashes was accepted")
			}
			if !strings.Contains(err.Error(), "backslash") {
				t.Errorf("the refusal does not say what is wrong: %v", err)
			}
		})
	}
}

// The positive sibling: the same items written with "/" are accepted, so the
// rule refuses the backslash and not the path.
func TestTheSamePathsWrittenWithSlashesAreAccepted(t *testing.T) {
	body := `version: 1
items:
  - id: skill/x
    kind: skill
    source: skills/x
    target: ~/.claude/skills/x
`
	if _, err := manifest.Parse([]byte(body), t.TempDir()); err != nil {
		t.Fatalf("a manifest written with forward slashes was refused: %v", err)
	}
}
