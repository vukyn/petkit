package link_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/link"
	"github.com/vukyn/petkit/internal/manifest"
)

// machine is a repository and a home directory made out of t.TempDir(). The
// home directory is a value passed into every call — nothing here reads $HOME.
type machine struct {
	t    *testing.T
	root string
	home string
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	base := t.TempDir()
	m := &machine{
		t:    t,
		root: filepath.Join(base, "petkit"),
		home: filepath.Join(base, "home"),
	}
	mustMkdirAll(t, m.root)
	mustMkdirAll(t, m.home)
	return m
}

// skill writes a source directory with one file in it, so a test can prove a
// directory survived by reading its contents rather than by stat-ing the path.
func (m *machine) skill(name, contents string) {
	m.t.Helper()
	dir := filepath.Join(m.root, "skills", name)
	mustMkdirAll(m.t, dir)
	mustWrite(m.t, filepath.Join(dir, "SKILL.md"), contents)
}

func (m *machine) manifestWith(items ...manifest.Item) *manifest.Manifest {
	m.t.Helper()
	loaded := &manifest.Manifest{Version: manifest.SupportedVersion, Items: items, Root: m.root}
	if err := loaded.Validate(); err != nil {
		m.t.Fatalf("fixture manifest is invalid: %v", err)
	}
	return loaded
}

func skillItem(name string) manifest.Item {
	return manifest.Item{
		ID:     "skill/" + name,
		Kind:   "skill",
		Source: "skills/" + name,
		Target: "~/.claude/skills/" + name,
	}
}

func (m *machine) targetOf(name string) string {
	return filepath.Join(m.home, ".claude", "skills", name)
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustSymlink(t *testing.T, destination, path string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.Symlink(destination, path); err != nil {
		t.Fatalf("symlink %s -> %s: %v", path, destination, err)
	}
}

// snapshot records every path under a directory together with what it is and,
// for files and symlinks, what it holds — enough to prove nothing changed.
func snapshot(t *testing.T, root string) []string {
	t.Helper()
	var lines []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			destination, readErr := os.Readlink(path)
			if readErr != nil {
				return readErr
			}
			lines = append(lines, fmt.Sprintf("symlink %s -> %s", relative, destination))
		case info.IsDir():
			lines = append(lines, fmt.Sprintf("dir     %s", relative))
		default:
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			lines = append(lines, fmt.Sprintf("file    %s %q %s", relative, content, info.Mode()))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	sort.Strings(lines)
	return lines
}

func syncOnce(t *testing.T, loaded *manifest.Manifest, home string, dryRun bool) link.Result {
	t.Helper()
	result, err := link.Apply(link.Plan(link.Inspect(loaded, home)), dryRun)
	if err != nil {
		t.Fatalf("sync returned an error it should not have: %v", err)
	}
	return result
}

// A real directory at a target is never removed: sync refuses the item, names
// it, and the directory is still there with its contents afterwards.
func TestSyncRefusesAConflictAndLeavesTheDirectoryWithItsContents(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "from the repository")
	loaded := m.manifestWith(skillItem("writing-todo"))

	target := m.targetOf("writing-todo")
	mustWrite(t, filepath.Join(target, "SKILL.md"), "somebody else's work")
	mustWrite(t, filepath.Join(target, "notes", "deep.md"), "nested and precious")

	states := link.Inspect(loaded, m.home)
	if states[0].Status != link.Conflict {
		t.Fatalf("status = %q, want %q", states[0].Status, link.Conflict)
	}

	result := syncOnce(t, loaded, m.home, false)
	if len(result.Refused) != 1 {
		t.Fatalf("refused %d items, want 1", len(result.Refused))
	}
	if result.Refused[0].State.Item.ID != "skill/writing-todo" {
		t.Errorf("refusal names %q, want the item id", result.Refused[0].State.Item.ID)
	}
	if result.Changed != 0 {
		t.Errorf("changed %d items, want 0", result.Changed)
	}

	// The point of the test: the directory is not merely still a path.
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("the conflicting directory is gone: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("the conflicting directory was replaced by a symlink")
	}
	assertFile(t, filepath.Join(target, "SKILL.md"), "somebody else's work")
	assertFile(t, filepath.Join(target, "notes", "deep.md"), "nested and precious")
}

// The failing case of the guard above: with nothing in the way the very same
// item links, so the refusal is a decision and not an inability.
func TestSyncLinksTheSameItemWhenNothingIsInTheWay(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "from the repository")
	loaded := m.manifestWith(skillItem("writing-todo"))

	result := syncOnce(t, loaded, m.home, false)
	if result.Changed != 1 || len(result.Refused) != 0 {
		t.Fatalf("changed=%d refused=%d, want 1 and 0", result.Changed, len(result.Refused))
	}
	assertSymlinkResolvesTo(t, m.targetOf("writing-todo"), filepath.Join(m.root, "skills", "writing-todo"))
	assertFile(t, filepath.Join(m.targetOf("writing-todo"), "SKILL.md"), "from the repository")
}

// A second sync reports no changes: the tool can be run twice.
func TestSyncIsIdempotent(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	m.skill("todo-checklist", "two")
	loaded := m.manifestWith(skillItem("writing-todo"), skillItem("todo-checklist"))

	first := syncOnce(t, loaded, m.home, false)
	if first.Changed != 2 {
		t.Fatalf("first sync changed %d items, want 2", first.Changed)
	}

	afterFirst := snapshot(t, m.home)

	second := syncOnce(t, loaded, m.home, false)
	if second.Changed != 0 {
		t.Errorf("second sync changed %d items, want 0", second.Changed)
	}
	if len(second.Refused) != 0 {
		t.Errorf("second sync refused %d items, want 0", len(second.Refused))
	}
	for _, itemState := range link.Inspect(loaded, m.home) {
		if itemState.Status != link.Linked {
			t.Errorf("item %q is %q after two syncs, want %q", itemState.Item.ID, itemState.Status, link.Linked)
		}
	}
	assertSnapshotsEqual(t, afterFirst, snapshot(t, m.home))
}

// A symlink pointing at the wrong place is repointed, and the new link resolves
// to the source the manifest names.
func TestSyncRepointsASymlinkThatPointsAtTheWrongPlace(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "the real one")
	elsewhere := filepath.Join(m.home, "elsewhere", "writing-todo")
	mustWrite(t, filepath.Join(elsewhere, "SKILL.md"), "an older copy")

	loaded := m.manifestWith(skillItem("writing-todo"))
	target := m.targetOf("writing-todo")
	mustSymlink(t, elsewhere, target)

	states := link.Inspect(loaded, m.home)
	if states[0].Status != link.Stale {
		t.Fatalf("status = %q, want %q", states[0].Status, link.Stale)
	}
	if states[0].Actual != elsewhere {
		t.Errorf("stale link reports %q, want %q", states[0].Actual, elsewhere)
	}

	result := syncOnce(t, loaded, m.home, false)
	if result.Changed != 1 {
		t.Fatalf("changed %d items, want 1", result.Changed)
	}

	source := filepath.Join(m.root, "skills", "writing-todo")
	assertSymlinkResolvesTo(t, target, source)
	assertFile(t, filepath.Join(target, "SKILL.md"), "the real one")

	// Repointing removes a symlink and nothing else: what it pointed at stays.
	assertFile(t, filepath.Join(elsewhere, "SKILL.md"), "an older copy")
}

// --dry-run changes nothing at all: the filesystem before equals the one after.
func TestDryRunChangesNothing(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	m.skill("todo-checklist", "two")
	m.skill("conflicted", "three")
	loaded := m.manifestWith(
		skillItem("writing-todo"),
		skillItem("todo-checklist"),
		skillItem("conflicted"),
	)

	// One of each interesting shape: missing, stale, conflict.
	mustSymlink(t, filepath.Join(m.home, "elsewhere"), m.targetOf("todo-checklist"))
	mustWrite(t, filepath.Join(m.targetOf("conflicted"), "SKILL.md"), "not ours")

	before := snapshot(t, m.home)
	beforeRepo := snapshot(t, m.root)

	result := syncOnce(t, loaded, m.home, true)
	if result.Changed != 2 {
		t.Errorf("dry run reports %d change(s), want the 2 it would make", result.Changed)
	}
	if len(result.Refused) != 1 {
		t.Errorf("dry run refuses %d, want 1", len(result.Refused))
	}

	assertSnapshotsEqual(t, before, snapshot(t, m.home))
	assertSnapshotsEqual(t, beforeRepo, snapshot(t, m.root))
}

// The failing case of the dry-run guard: the identical plan, run for real,
// does change the filesystem — so "nothing changed" means restraint.
func TestSyncWithoutDryRunDoesChangeTheFilesystem(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	loaded := m.manifestWith(skillItem("writing-todo"))

	before := snapshot(t, m.home)
	syncOnce(t, loaded, m.home, false)
	after := snapshot(t, m.home)

	if len(before) == len(after) {
		t.Fatalf("the real run changed nothing: before=%v after=%v", before, after)
	}
}

// Creating a link creates the directories above it.
func TestSyncCreatesParentDirectories(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	loaded := m.manifestWith(manifest.Item{
		ID:     "skill/writing-todo",
		Source: "skills/writing-todo",
		Target: "~/.claude/deeply/nested/skills/writing-todo",
	})

	if _, err := os.Stat(filepath.Join(m.home, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("the fixture already has ~/.claude; the test would prove nothing")
	}
	syncOnce(t, loaded, m.home, false)
	assertSymlinkResolvesTo(t,
		filepath.Join(m.home, ".claude", "deeply", "nested", "skills", "writing-todo"),
		filepath.Join(m.root, "skills", "writing-todo"))
}

// A regular file in the way is a conflict for the same reason a directory is.
func TestARegularFileAtATargetIsAConflict(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	loaded := m.manifestWith(skillItem("writing-todo"))
	mustWrite(t, m.targetOf("writing-todo"), "a file, not a directory")

	states := link.Inspect(loaded, m.home)
	if states[0].Status != link.Conflict {
		t.Fatalf("status = %q, want %q", states[0].Status, link.Conflict)
	}
	if !strings.Contains(states[0].Detail, "regular file") {
		t.Errorf("detail = %q, want it to say what is in the way", states[0].Detail)
	}
	syncOnce(t, loaded, m.home, false)
	assertFile(t, m.targetOf("writing-todo"), "a file, not a directory")
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(content) != want {
		t.Errorf("%s contains %q, want %q", path, content, want)
	}
}

func assertSymlinkResolvesTo(t *testing.T, path, want string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve %s: %v", path, err)
	}
	wantResolved, err := filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatalf("resolve %s: %v", want, err)
	}
	if resolved != wantResolved {
		t.Errorf("%s resolves to %s, want %s", path, resolved, wantResolved)
	}
}

func assertSnapshotsEqual(t *testing.T, before, after []string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("the filesystem changed:\nbefore %v\nafter  %v", before, after)
	}
	for index := range before {
		if before[index] != after[index] {
			t.Errorf("the filesystem changed:\nbefore %s\nafter  %s", before[index], after[index])
		}
	}
}
