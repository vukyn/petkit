package link

// ⚠️ This is the one test file in the package that is `package link` rather than
// `package link_test`, and the reason is worth writing down: doctor's survey
// reads a real directory, and a Windows layout resolved on this machine produces
// paths this filesystem cannot hold in the shape Windows would. The decision
// underneath — "is this entry one the manifest claims?" — is pure, so it is
// measured directly rather than through a filesystem that cannot stage it.

import (
	"testing"

	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
)

func windowsLayout() manifest.Layout {
	return manifest.Layout{
		Home:   `C:\Users\me`,
		Config: `C:\Users\me\.claude`,
		GOOS:   ospath.Windows,
	}
}

func writingTodo() manifest.Item {
	return manifest.Item{
		ID:     "skill/writing-todo",
		Kind:   "skill",
		Source: "skills/writing-todo",
		Target: "~/.claude/skills/writing-todo",
	}
}

// On Windows the name read out of the skills directory carries whatever case it
// was created with, and the manifest carries its own. They are one file, so
// doctor has to recognise its own item in either spelling — otherwise it reports
// a skill petkit installed as a stranger petkit leaves alone.
func TestOnWindowsAManagedTargetIsRecognisedInAnyCase(t *testing.T) {
	layout := windowsLayout()
	managed := newManagedTargets(1, layout.GOOS)
	managed.claim(writingTodo(), layout)

	for _, spelling := range []string{
		`C:\Users\me\.claude\skills\writing-todo`,
		`C:\Users\me\.claude\skills\Writing-Todo`,
		`c:\users\me\.claude\skills\WRITING-TODO`,
		`C:/Users/me/.claude/skills/writing-todo`,
	} {
		if _, found := managed.owner(spelling); !found {
			t.Errorf("doctor would call %s a stranger, and petkit installed it", spelling)
		}
	}
	if _, found := managed.owner(`C:\Users\me\.claude\skills\somebody-else`); found {
		t.Error("an entry petkit does not manage was claimed")
	}
}

// The sibling: off Windows those spellings are different files, and doctor must
// keep saying so — a fold there would hide a genuinely unmanaged directory.
func TestEverywhereElseAManagedTargetIsRecognisedByItsExactName(t *testing.T) {
	layout := manifest.Layout{Home: "/home/me", Config: "/home/me/.claude", GOOS: "linux"}
	managed := newManagedTargets(1, layout.GOOS)
	managed.claim(writingTodo(), layout)

	if _, found := managed.owner("/home/me/.claude/skills/writing-todo"); !found {
		t.Error("doctor did not recognise its own item by its exact name")
	}
	if _, found := managed.owner("/home/me/.claude/skills/Writing-Todo"); found {
		t.Error("a differently-named Unix directory was claimed as petkit's")
	}
}
