package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
)

func parse(t *testing.T, body string) (*manifest.Manifest, error) {
	t.Helper()
	return manifest.Parse([]byte(body), t.TempDir())
}

const oneGoodItem = `
version: 1
items:
  - id: skill/writing-todo
    kind: skill
    source: skills/writing-todo
    target: ~/.claude/skills/writing-todo
`

// A manifest that says everything correctly is accepted — without this, every
// refusal below could be an accident.
func TestAValidManifestIsAccepted(t *testing.T) {
	parsed, err := parse(t, oneGoodItem)
	if err != nil {
		t.Fatalf("a valid manifest was refused: %v", err)
	}
	if len(parsed.Items) != 1 {
		t.Fatalf("parsed %d items, want 1", len(parsed.Items))
	}
	if parsed.Items[0].Target != "~/.claude/skills/writing-todo" {
		t.Errorf("target = %q", parsed.Items[0].Target)
	}
}

// An absolute target is refused, and the message names the item, because an
// absolute target would make petkit.yaml machine-specific.
func TestAnAbsoluteTargetIsRefusedAndNamesTheItem(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/writing-todo
    source: skills/writing-todo
    target: /Users/someone/.claude/skills/writing-todo
`)
	if err == nil {
		t.Fatal("an absolute target was accepted")
	}
	if !strings.Contains(err.Error(), "skill/writing-todo") {
		t.Errorf("error does not name the item: %v", err)
	}
	if !strings.Contains(err.Error(), "/Users/someone/.claude/skills/writing-todo") {
		t.Errorf("error does not quote the offending target: %v", err)
	}
}

// A relative target is refused for the same reason.
func TestARelativeTargetIsRefused(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/writing-todo
    source: skills/writing-todo
    target: .claude/skills/writing-todo
`)
	if err == nil {
		t.Fatal("a relative target was accepted")
	}
	if !strings.Contains(err.Error(), "skill/writing-todo") {
		t.Errorf("error does not name the item: %v", err)
	}
}

// Two items claiming one target is refused: whichever ran last would win, and
// the manifest would describe a machine that cannot exist.
func TestTwoItemsSharingATargetAreRefused(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/first
    source: skills/first
    target: ~/.claude/skills/todo
  - id: skill/second
    source: skills/second
    target: ~/.claude/skills/todo
`)
	if err == nil {
		t.Fatal("a duplicated target was accepted")
	}
	for _, name := range []string{"skill/first", "skill/second", "~/.claude/skills/todo"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error does not name %q: %v", name, err)
		}
	}

	// The failing case: the same two items with distinct targets are accepted.
	if _, err := parse(t, `
version: 1
items:
  - id: skill/first
    source: skills/first
    target: ~/.claude/skills/first
  - id: skill/second
    source: skills/second
    target: ~/.claude/skills/second
`); err != nil {
		t.Fatalf("two items with distinct targets were refused: %v", err)
	}
}

func TestADuplicatedIdIsRefused(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/same
    source: skills/first
    target: ~/.claude/skills/first
  - id: skill/same
    source: skills/second
    target: ~/.claude/skills/second
`)
	if err == nil {
		t.Fatal("a duplicated id was accepted")
	}
	if !strings.Contains(err.Error(), "skill/same") {
		t.Errorf("error does not name the id: %v", err)
	}
}

// A source that climbs out of the repository is refused: the repository is the
// only copy, so a source outside it is not petkit's to install.
func TestASourceThatEscapesTheRepositoryIsRefused(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/escapee
    source: ../../elsewhere/skills/x
    target: ~/.claude/skills/x
`)
	if err == nil {
		t.Fatal("an escaping source was accepted")
	}
	if !strings.Contains(err.Error(), "skill/escapee") {
		t.Errorf("error does not name the item: %v", err)
	}

	// The failing case: a source with a .. that stays inside is accepted.
	if _, err := parse(t, `
version: 1
items:
  - id: skill/inside
    source: skills/nested/../writing-todo
    target: ~/.claude/skills/writing-todo
`); err != nil {
		t.Fatalf("a source that stays inside the repository was refused: %v", err)
	}
}

func TestAnAbsoluteSourceIsRefused(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/absolute
    source: /etc/skills/x
    target: ~/.claude/skills/x
`)
	if err == nil {
		t.Fatal("an absolute source was accepted")
	}
	if !strings.Contains(err.Error(), "skill/absolute") {
		t.Errorf("error does not name the item: %v", err)
	}
}

// MFST-006. ⚠️ The test above is not evidence on its own, and believing it was
// cost a release: it names one Unix spelling, and a Unix spelling was refused
// on macOS by the host's own filepath.IsAbs while being accepted on Windows.
// Every spelling that is absolute *somewhere* is named here, so the answer
// cannot come from the machine running the suite.
//
// Its positive case is TestASourceThatEscapesTheRepositoryIsRefused above: a
// relative source, `..` and all, is still accepted.
func TestASourceAbsoluteOnAnyPlatformIsRefused(t *testing.T) {
	for _, source := range []string{
		"/etc/skills/x",    // absolute on Unix; filepath.IsAbs says no on Windows
		"C:/skills/x",      // absolute on Windows; filepath.IsAbs says no on Unix
		`C:\skills\x`,      // the same, spelled the way Windows spells it
		`\\server\share\x`, // a UNC share
	} {
		_, err := parse(t, "version: 1\nitems:\n  - id: skill/absolute\n"+
			"    source: "+source+"\n    target: ~/.claude/skills/x\n")
		if err == nil {
			t.Errorf("source %q was accepted", source)
			continue
		}
		if !strings.Contains(err.Error(), "skill/absolute") {
			t.Errorf("source %q: error does not name the item: %v", source, err)
		}
	}
}

func TestAnUnknownVersionIsRefused(t *testing.T) {
	_, err := parse(t, strings.Replace(oneGoodItem, "version: 1", "version: 7", 1))
	if err == nil {
		t.Fatal("an unknown schema version was accepted")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error does not mention the version: %v", err)
	}
}

// Every problem is reported at once, so a bad manifest takes one run to fix.
func TestAllProblemsAreReportedTogether(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/one
    source: /absolute
    target: /also-absolute
  - id: skill/one
    source: skills/two
    target: ~/.claude/skills/two
`)
	if err == nil {
		t.Fatal("a manifest with three problems was accepted")
	}
	problems := strings.Count(err.Error(), "\n  - ")
	if problems != 3 {
		t.Errorf("reported %d problems, want 3 (absolute target, absolute source, duplicate id): %v", problems, err)
	}
}

// ~ is expanded against the home directory it is given, and nothing else is
// expanded at all.
func TestTildeIsTheOnlyTemplating(t *testing.T) {
	home := "/tmp/home"
	cases := map[string]string{
		"~/.claude/skills/x": filepath.Join(home, ".claude", "skills", "x"),
		"~":                  home,
		"$HOME/.claude":      "$HOME/.claude",
		"${HOME}/.claude":    "${HOME}/.claude",
	}
	for input, want := range cases {
		if got := manifest.ExpandTilde(input, home); got != want {
			t.Errorf("ExpandTilde(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDisplayIsTheInverseOfExpansion(t *testing.T) {
	home := "/tmp/home"
	layout := manifest.NewLayout(home, ospath.Current(), nil)
	expanded := layout.Resolve("~/.claude/skills/x")
	if got := layout.Display(expanded); got != "~/.claude/skills/x" {
		t.Errorf("Display(%q) = %q", expanded, got)
	}
	if got := layout.Display("/elsewhere/x"); got != "/elsewhere/x" {
		t.Errorf("a path outside the home directory was collapsed to %q", got)
	}
}

// The repository's own petkit.yaml has to pass its own validator; it is the
// only manifest that ships with the tool.
func TestTheRepositoryManifestIsValid(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve the repository root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, manifest.FileName)); err != nil {
		t.Skipf("no %s next to the test: %v", manifest.FileName, err)
	}
	loaded, err := manifest.Load(root)
	if err != nil {
		t.Fatalf("the repository's own manifest is invalid: %v", err)
	}
	for _, item := range loaded.Items {
		if _, err := os.Stat(item.SourcePath(root, ospath.Current())); err != nil {
			t.Errorf("item %q names a source that is not in the repository: %v", item.ID, err)
		}
	}
}

// `~/` is a prefix and not a fence. A target may start with it and still leave
// the home directory once expanded, and sync would have created that link before
// doctor ever ran — so validation is where it has to be caught.
func TestATargetThatClimbsOutOfHomeIsRefused(t *testing.T) {
	_, err := parse(t, `
version: 1
items:
  - id: skill/writing-todo
    source: skills/writing-todo
    target: ~/../elsewhere/writing-todo
`)
	if err == nil {
		t.Fatal("a target above the home directory was accepted")
	}
	if !strings.Contains(err.Error(), "skill/writing-todo") {
		t.Errorf("error does not name the item: %v", err)
	}
	if !strings.Contains(err.Error(), "~/../elsewhere/writing-todo") {
		t.Errorf("error does not quote the offending target: %v", err)
	}
}

// Its positive case: a target that uses `..` and still lands inside home is
// accepted, so the guard above refuses the escape and not the two dots.
func TestATargetWithDotsThatStaysUnderHomeIsAccepted(t *testing.T) {
	m, err := parse(t, `
version: 1
items:
  - id: skill/writing-todo
    source: skills/writing-todo
    target: ~/.claude/skills/../skills/writing-todo
`)
	if err != nil {
		t.Fatalf("a target that stays under home was refused: %v", err)
	}
	if len(m.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(m.Items))
	}
}
