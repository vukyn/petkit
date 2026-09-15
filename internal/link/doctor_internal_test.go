package link

// ⚠️ This is the one test file in the package that is `package link` rather than
// `package link_test`, and the reason is worth writing down: doctor's survey
// reads a real directory, and a Windows layout resolved on this machine produces
// paths this filesystem cannot hold in the shape Windows would. The decision
// underneath — "is this entry one the manifest claims?" — is pure, so it is
// measured directly rather than through a filesystem that cannot stage it.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
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

// LINK-006. ⚠️ os.Stat failing is not the same fact as the destination being
// absent, and doctor printed the second sentence for both. On Windows a
// file-flavour symlink pointing at a directory fails Stat with `Access is
// denied.` and os.IsNotExist(err) is **false** — so doctor reported a path as
// not existing while it existed, run after run, and the reading was never going
// to improve on its own.
//
// ⚠️ Stat is a parameter here for the same reason the platform is one elsewhere:
// the error Windows produces cannot be made to happen on the filesystem this
// test runs on, so it is handed in instead of staged.
func TestAFailedFollowIsNotTheSameFactAsAMissingDestination(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "destination")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(destination, link); err != nil {
		t.Fatalf("the fixture needs a symlink: %v", err)
	}

	cases := []struct {
		name string
		stat statFunc
		want symlinkState
	}{
		{
			name: "the destination is there",
			stat: os.Stat,
			want: followed,
		},
		{
			name: "the destination is gone",
			stat: failWith(&os.PathError{Op: "CreateFile", Path: link, Err: fs.ErrNotExist}),
			want: dangling,
		},
		{
			name: "Windows would not open it",
			stat: failWith(&os.PathError{Op: "CreateFile", Path: link, Err: errors.New("Access is denied.")}),
			want: unfollowable,
		},
		{
			name: "the directory above it is closed",
			stat: failWith(&os.PathError{Op: "CreateFile", Path: link, Err: fs.ErrPermission}),
			want: unfollowable,
		},
	}
	for _, c := range cases {
		state, got, _ := followSymlink(link, c.stat)
		if state != c.want {
			t.Errorf("%s: state = %v, want %v", c.name, state, c.want)
		}
		if state != followed && got != destination {
			t.Errorf("%s: destination = %q, want %q", c.name, got, destination)
		}
	}
}

// The positive case the classification owes: something that is not a symlink at
// all is never any of the three, whatever Stat says about it.
func TestARealDirectoryIsNotASymlinkInAnyState(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain")
	if err := os.Mkdir(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, stat := range []statFunc{os.Stat, failWith(fs.ErrNotExist), failWith(fs.ErrPermission)} {
		if state, _, _ := followSymlink(plain, stat); state != notASymlink {
			t.Errorf("a real directory was classified as %v", state)
		}
	}
}

func failWith(err error) statFunc {
	return func(string) (os.FileInfo, error) { return nil, err }
}

// The message names one path, and it is the destination. ⚠️ os.Stat wraps its
// error in an *fs.PathError carrying the path Stat was given — the link, not the
// destination — so quoting the error whole made doctor say "it points at X, and
// following it failed with CreateFile Y: ...", where X and Y are different paths
// and a reader has to work out that neither reading is wrong. The reason on its
// own is the part that is news.
func TestTheReasonAFollowFailedIsQuotedWithoutItsPath(t *testing.T) {
	wrapped := &os.PathError{Op: "CreateFile", Path: `C:\link`, Err: errors.New("Access is denied.")}
	if got := reason(wrapped); got != "Access is denied." {
		t.Errorf("reason = %q, want the error without its path", got)
	}
	// The positive case: an error that is not about a path is quoted whole.
	if got := reason(errors.New("something else")); got != "something else" {
		t.Errorf("reason = %q, want the error itself", got)
	}
}
