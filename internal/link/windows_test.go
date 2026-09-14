package link_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/link"
	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
)

// Windows refuses os.Symlink with ERROR_PRIVILEGE_NOT_HELD unless Developer
// Mode is on or the process is elevated. Untranslated it arrives as a syscall
// error that names neither cause nor cure.
//
// ⚠️ The error is built here rather than provoked, which is the point: the
// translation is measured on the machine that CANNOT produce the failure, so it
// is measured at all. os.Symlink wraps its errno in *os.LinkError, so the
// synthetic error carries the same shape the real one does.
func TestTheWindowsSymlinkPrivilegeFailureIsExplained(t *testing.T) {
	refused := &os.LinkError{
		Op:  "symlink",
		Old: `C:\Users\me\.petkit\skills\writing-todo`,
		New: `C:\Users\me\.claude\skills\writing-todo`,
		Err: link.ErrWindowsPrivilegeNotHeld,
	}

	message := link.ExplainSymlinkFailure(refused, "skill/writing-todo", refused.New)
	if message == "" {
		t.Fatal("the privilege failure was not recognised, so it would surface as a raw syscall error")
	}
	for _, want := range []string{
		"skill/writing-todo",                      // which item
		`C:\Users\me\.claude\skills\writing-todo`, // which path
		"ERROR_PRIVILEGE_NOT_HELD",                // the cause, by the name the web will match
		"Developer Mode",                          // the first way out
		"Administrator",                           // the second
		"petkit sync",                             // what to run afterwards
	} {
		if !strings.Contains(message, want) {
			t.Errorf("the message does not mention %q:\n%s", want, message)
		}
	}
	// The one thing it must never offer. A copy would let the installed skill
	// and the repository drift, which is the failure petkit exists to prevent.
	if strings.Contains(message, "will copy") || strings.Contains(message, "copying instead") {
		t.Errorf("the message offers a copy as a way out:\n%s", message)
	}
}

// The positive sibling: every other failure keeps the message it had, which is
// what makes this a translation rather than a rewrite.
func TestAnOrdinarySymlinkFailureIsNotTranslated(t *testing.T) {
	if message := link.ExplainSymlinkFailure(errors.New("no space left on device"), "skill/x", "/home/me/x"); message != "" {
		t.Errorf("an ordinary error was translated as a privilege failure:\n%s", message)
	}
	if message := link.ExplainSymlinkFailure(nil, "skill/x", "/home/me/x"); message != "" {
		t.Errorf("a nil error produced a message:\n%s", message)
	}
}

// On Windows a symlink destination that differs from the source only in case
// points at the same directory, so the item is `linked` and sync owes it
// nothing.
//
// ⚠️ The status is only half of what is being measured. A byte comparison
// reports `stale`, and `sync` then REMOVES the symlink and recreates it — the
// one destructive operation petkit has, firing for no reason, on every run. The
// plan is asserted alongside the status for exactly that reason.
func TestOnWindowsALinkThatDiffersOnlyInCaseIsLeftAlone(t *testing.T) {
	loaded, layout := caseDifferingMachine(t, ospath.Windows)

	states := link.Inspect(loaded, layout)
	if states[0].Status != link.Linked {
		t.Fatalf("status on Windows = %q, want %q (%s -> %s)",
			states[0].Status, link.Linked, states[0].Target, states[0].Actual)
	}
	if kind := link.Plan(states)[0].Kind; kind != link.None {
		t.Errorf("sync would %s a link that is already correct on Windows", kind)
	}
}

// The sibling, and the reason the fold is not unconditional: off Windows those
// are two different directories and the link really is stale.
func TestEverywhereElseALinkThatDiffersOnlyInCaseIsStale(t *testing.T) {
	loaded, layout := caseDifferingMachine(t, "linux")

	states := link.Inspect(loaded, layout)
	if states[0].Status != link.Stale {
		t.Fatalf("status off Windows = %q, want %q (%s -> %s)",
			states[0].Status, link.Stale, states[0].Target, states[0].Actual)
	}
	if kind := link.Plan(states)[0].Kind; kind != link.Repoint {
		t.Errorf("sync would %s a link that points somewhere else", kind)
	}
}

// caseDifferingMachine is one item whose installed symlink points at its source
// spelled in another case. The source exists as `skills/Writing-Todo`; the link
// says `skills/writing-todo`.
//
// The fixture is built so that it reads the same on a case-sensitive filesystem
// and a case-insensitive one: on a case-sensitive volume the destination does
// not exist and symlink resolution fails, on a case-insensitive one it resolves
// to the spelling it was given. Either way the two paths differ as strings, so
// the only thing that can make them one place is the case fold.
//
// ⚠️ The link is created at whatever path the layout resolves the target to,
// not at a path spelled here. A Windows layout converts the manifest's "/" to a
// backslash, so on this machine the link's NAME contains a backslash — which is
// legal on macOS and is exactly what makes the Windows resolution watchable
// here. Spelling the path independently would measure the fixture instead.
func caseDifferingMachine(t *testing.T, goos string) (*manifest.Manifest, manifest.Layout) {
	t.Helper()
	m := newMachine(t)
	m.skill("Writing-Todo", "from the repository")

	loaded := m.manifestWith(manifest.Item{
		ID:     "skill/writing-todo",
		Kind:   "skill",
		Source: "skills/Writing-Todo",
		Target: "~/.claude/skills/writing-todo",
	})
	layout := m.layoutFor(goos)

	target := loaded.Items[0].TargetPath(layout)
	mustMkdirAll(t, filepath.Dir(target))
	destination := filepath.Join(m.root, "skills", "writing-todo")
	if err := os.Symlink(destination, target); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	return loaded, layout
}
