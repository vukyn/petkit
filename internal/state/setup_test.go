package state_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/state"
)

// gitLog is a git that never runs git: it records what it was asked for, and
// answers `clone` by writing a repository at the path it was given. Nothing in
// these tests reaches the network.
type gitLog struct {
	calls [][]string
}

func (l *gitLog) runner(t *testing.T) state.Runner {
	t.Helper()
	return func(_ string, args ...string) (string, error) {
		l.calls = append(l.calls, args)
		if len(args) == 3 && args[0] == "clone" {
			repoAt(t, args[2])
		}
		return "", nil
	}
}

// The URL is derived from the module the binary was built from, so a fork
// installed from its own module path clones itself. The assertion is on the
// argument the runner received rather than on anything printed: a constant
// smuggled back in would still print a plausible line.
func TestTheCloneURLIsBuiltFromTheModulePath(t *testing.T) {
	base := t.TempDir()
	log := &gitLog{}

	result, err := state.Setup(state.SetupRequest{
		Path:       filepath.Join(base, "clone"),
		Home:       filepath.Join(base, "home"),
		ModulePath: "example.com/someone/petkit",
		Run:        log.runner(t),
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	if len(log.calls) != 1 {
		t.Fatalf("git was called %d time(s), want once: %v", len(log.calls), log.calls)
	}
	call := log.calls[0]
	if call[0] != "clone" {
		t.Errorf("git was asked for %q, want clone", call[0])
	}
	if call[1] != "https://example.com/someone/petkit" {
		t.Errorf("git was asked to clone %q, want https://example.com/someone/petkit", call[1])
	}
	if result.CloneURL != "https://example.com/someone/petkit" {
		t.Errorf("result reports %q as the source", result.CloneURL)
	}

	// No --depth anywhere: `petkit check` compares tags, and a shallow clone
	// arrives without them.
	for _, argument := range call {
		if strings.HasPrefix(argument, "--depth") {
			t.Errorf("the clone is shallow (%q), so check would have no tags", argument)
		}
	}
}

// A build that carries no module path cannot know where it came from, and says
// so with the way out in the same sentence.
func TestABuildWithNoModulePathIsRefusedAndNamesTheManualClone(t *testing.T) {
	base := t.TempDir()
	log := &gitLog{}

	_, err := state.Setup(state.SetupRequest{
		Path:       filepath.Join(base, "clone"),
		Home:       filepath.Join(base, "home"),
		ModulePath: "",
		Run:        log.runner(t),
	})
	if err == nil {
		t.Fatal("setup ran with no module path to clone from")
	}
	if !strings.Contains(err.Error(), "git clone") {
		t.Errorf("the error does not name the way out: %v", err)
	}
	if len(log.calls) != 0 {
		t.Errorf("git ran anyway: %v", log.calls)
	}
}

// The rule sync lives by, applied to the one directory setup writes: a target
// with anything in it is refused by name, and what is in it is still there
// afterwards — read back, not stat'ed.
func TestSetupRefusesANonEmptyTargetAndLeavesItAlone(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "occupied")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	kept := filepath.Join(target, "notes.md")
	if err := os.WriteFile(kept, []byte("somebody else's work"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	log := &gitLog{}

	_, err := state.Setup(state.SetupRequest{
		Path:       target,
		Home:       filepath.Join(base, "home"),
		ModulePath: "example.com/someone/petkit",
		Run:        log.runner(t),
	})
	if err == nil {
		t.Fatal("setup cloned into a directory that already had something in it")
	}
	if !strings.Contains(err.Error(), target) {
		t.Errorf("the refusal does not name the directory: %v", err)
	}
	if len(log.calls) != 0 {
		t.Errorf("git ran over an occupied target: %v", log.calls)
	}

	content, err := os.ReadFile(kept)
	if err != nil {
		t.Fatalf("the file in the target did not survive: %v", err)
	}
	if string(content) != "somebody else's work" {
		t.Errorf("the file in the target says %q now", content)
	}
	if _, err := os.Stat(state.ConfigPath(filepath.Join(base, "home"))); !os.IsNotExist(err) {
		t.Error("the refused setup recorded a repository anyway")
	}
}

// The positive case the refusal owes, twice over: an empty directory and a path
// that does not exist are both cloned into, so the test above measures what is
// in the directory and not the fact that it was named.
func TestSetupAcceptsAnEmptyDirectoryAndAMissingPath(t *testing.T) {
	cases := []struct {
		name     string
		relative []string
		prepare  func(t *testing.T, target string)
	}{
		{
			name:     "an empty directory",
			relative: []string{"clone"},
			prepare: func(t *testing.T, target string) {
				if err := os.MkdirAll(target, 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
			},
		},
		{
			name:     "a path that does not exist",
			relative: []string{"clone"},
			prepare:  func(*testing.T, string) {},
		},
		{
			name:     "a path whose parent does not exist either",
			relative: []string{"deeper", "clone"},
			prepare:  func(*testing.T, string) {},
		},
	}
	for _, testCase := range cases {
		name := testCase.name
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			home := filepath.Join(base, "home")
			target := filepath.Join(append([]string{base}, testCase.relative...)...)
			testCase.prepare(t, target)
			log := &gitLog{}

			result, err := state.Setup(state.SetupRequest{
				Path:       target,
				Home:       home,
				ModulePath: "example.com/someone/petkit",
				Run:        log.runner(t),
			})
			if err != nil {
				t.Fatalf("setup refused %s: %v", name, err)
			}
			if result.Location.Root != target {
				t.Errorf("recorded %q, want %q", result.Location.Root, target)
			}

			// The record is init's, so the next command finds the clone from
			// anywhere — that is the whole point of setup.
			location, err := state.Find(base, home, noEnv)
			if err != nil {
				t.Fatalf("the clone is not findable after setup: %v", err)
			}
			if location.Root != target {
				t.Errorf("find answers %q, want %q", location.Root, target)
			}
		})
	}
}

// A machine that is already set up is refused rather than given a second copy
// of the thing whose whole point is that there is one — and the message says
// where the first one is.
func TestSetupRefusesAMachineThatAlreadyHasARepository(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	existing := repoAt(t, filepath.Join(base, "already"))
	if _, err := state.Init(existing, home); err != nil {
		t.Fatalf("init: %v", err)
	}
	log := &gitLog{}

	_, err := state.Setup(state.SetupRequest{
		Home:       home,
		ModulePath: "example.com/someone/petkit",
		Run:        log.runner(t),
	})
	if err == nil {
		t.Fatal("setup cloned a second copy onto a machine that already had one")
	}
	if !strings.Contains(err.Error(), existing) {
		t.Errorf("the refusal does not name the recorded repository: %v", err)
	}
	if len(log.calls) != 0 {
		t.Errorf("git ran anyway: %v", log.calls)
	}
}

// The positive cases the refusal owes. A path naming somewhere else is an
// explicit second clone and is allowed; and a recorded path that no longer
// resolves is not a set-up machine, so setup is one of the ways to fix it.
func TestSetupProceedsForAnotherPathAndForARecordThatNoLongerResolves(t *testing.T) {
	t.Run("a path naming somewhere else", func(t *testing.T) {
		base := t.TempDir()
		home := filepath.Join(base, "home")
		if _, err := state.Init(repoAt(t, filepath.Join(base, "already")), home); err != nil {
			t.Fatalf("init: %v", err)
		}
		log := &gitLog{}

		if _, err := state.Setup(state.SetupRequest{
			Path:       filepath.Join(base, "second"),
			Home:       home,
			ModulePath: "example.com/someone/petkit",
			Run:        log.runner(t),
		}); err != nil {
			t.Fatalf("setup refused an explicit second path: %v", err)
		}
		if len(log.calls) != 1 {
			t.Errorf("git was called %d time(s), want once", len(log.calls))
		}
	})

	t.Run("a record that no longer resolves", func(t *testing.T) {
		base := t.TempDir()
		home := filepath.Join(base, "home")
		gone := repoAt(t, filepath.Join(base, "gone"))
		if _, err := state.Init(gone, home); err != nil {
			t.Fatalf("init: %v", err)
		}
		if err := os.RemoveAll(gone); err != nil {
			t.Fatalf("remove: %v", err)
		}
		log := &gitLog{}

		if _, err := state.Setup(state.SetupRequest{
			Path:       filepath.Join(base, "fresh"),
			Home:       home,
			ModulePath: "example.com/someone/petkit",
			Run:        log.runner(t),
		}); err != nil {
			t.Fatalf("setup refused a machine whose record points at nothing: %v", err)
		}
	})
}

// A file where the clone should go is refused too, and for the same reason: it
// is not setup's to remove.
func TestSetupRefusesAFileWhereTheCloneWouldGo(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "petkit")
	if err := os.WriteFile(target, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	log := &gitLog{}

	if _, err := state.Setup(state.SetupRequest{
		Path:       target,
		Home:       filepath.Join(base, "home"),
		ModulePath: "example.com/someone/petkit",
		Run:        log.runner(t),
	}); err == nil {
		t.Fatal("setup cloned over a file")
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "not a directory" {
		t.Errorf("the file did not survive: %q %v", content, err)
	}
}

// With no path at all, setup clones to ~/.petkit — the default is a fact about
// the tool, so it is asserted rather than left to the prose.
func TestTheDefaultPathIsUnderTheHomeDirectory(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	log := &gitLog{}

	result, err := state.Setup(state.SetupRequest{
		Home:       home,
		ModulePath: "example.com/someone/petkit",
		Run:        log.runner(t),
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if want := filepath.Join(home, ".petkit"); result.Target != want {
		t.Errorf("setup cloned into %q, want %q", result.Target, want)
	}
}
