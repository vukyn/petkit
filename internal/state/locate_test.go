package state_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/state"
)

// noEnv is a machine with nothing in its environment; the environment is a
// parameter so a test never has to set a real variable.
func noEnv(string) string { return "" }

func envWith(name, value string) func(string) string {
	return func(key string) string {
		if key == name {
			return value
		}
		return ""
	}
}

func repoAt(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "petkit.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write the manifest: %v", err)
	}
	return dir
}

// Running anywhere inside the repository finds it, however deep.
func TestTheRepositoryIsFoundByWalkingUp(t *testing.T) {
	base := t.TempDir()
	root := repoAt(t, filepath.Join(base, "repo"))
	deep := filepath.Join(root, "internal", "link")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	location, err := state.Find(deep, filepath.Join(base, "home"), noEnv)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if location.Root != root {
		t.Errorf("root = %q, want %q", location.Root, root)
	}
}

// Outside the repository the environment variable answers.
func TestPetkitHomeAnswersWhenTheWorkingDirectoryIsElsewhere(t *testing.T) {
	base := t.TempDir()
	root := repoAt(t, filepath.Join(base, "repo"))
	elsewhere := filepath.Join(base, "elsewhere")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	location, err := state.Find(elsewhere, filepath.Join(base, "home"), envWith(state.EnvHome, root))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if location.Root != root {
		t.Errorf("root = %q, want %q", location.Root, root)
	}
	if location.From != state.EnvHome {
		t.Errorf("from = %q, want %q", location.From, state.EnvHome)
	}
}

// An environment variable that points at nothing is said out loud rather than
// quietly ignored: it was set on purpose.
func TestAWrongPetkitHomeIsReportedRatherThanSkipped(t *testing.T) {
	base := t.TempDir()
	elsewhere := filepath.Join(base, "elsewhere")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, err := state.Find(elsewhere, filepath.Join(base, "home"), envWith(state.EnvHome, filepath.Join(base, "nowhere")))
	if err == nil {
		t.Fatal("a PETKIT_HOME pointing at nothing was accepted")
	}
	if !strings.Contains(err.Error(), state.EnvHome) {
		t.Errorf("error does not name the variable: %v", err)
	}
}

// init records the repository, and the next command finds it from anywhere.
func TestInitRecordsTheRepositoryAndFindUsesIt(t *testing.T) {
	base := t.TempDir()
	root := repoAt(t, filepath.Join(base, "repo"))
	home := filepath.Join(base, "home")
	elsewhere := filepath.Join(base, "elsewhere")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := state.Find(elsewhere, home, noEnv); err == nil {
		t.Fatal("the repository was found before init recorded it")
	} else if !strings.Contains(err.Error(), "petkit init") {
		t.Errorf("the error does not say what to do: %v", err)
	}

	if _, err := state.Init(root, home); err != nil {
		t.Fatalf("init: %v", err)
	}

	location, err := state.Find(elsewhere, home, noEnv)
	if err != nil {
		t.Fatalf("find after init: %v", err)
	}
	if location.Root != root {
		t.Errorf("root = %q, want %q", location.Root, root)
	}

	var config state.Config
	content, err := os.ReadFile(state.ConfigPath(home))
	if err != nil {
		t.Fatalf("read the config: %v", err)
	}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("the config is not JSON: %v", err)
	}
	if config.Repo != root {
		t.Errorf("config records %q, want %q", config.Repo, root)
	}
}

// init refuses a path that is not a repository: a recorded path that resolves
// to nothing is worse than no record at all.
func TestInitRefusesAPathWithoutAManifest(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	notARepo := filepath.Join(base, "not-a-repo")
	if err := os.MkdirAll(notARepo, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := state.Init(notARepo, home); err == nil {
		t.Fatal("init accepted a directory with no manifest")
	} else if !strings.Contains(err.Error(), "petkit.yaml") {
		t.Errorf("the error does not say what is missing: %v", err)
	}
	if _, err := os.Stat(state.ConfigPath(home)); !os.IsNotExist(err) {
		t.Error("the refused init wrote a config anyway")
	}

	// The failing case: the same call against a real repository succeeds.
	if _, err := state.Init(repoAt(t, filepath.Join(base, "repo")), home); err != nil {
		t.Fatalf("init refused a real repository: %v", err)
	}
}

// A second init keeps the previous record beside the new one.
func TestInitBacksUpAConfigItReplaces(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	first := repoAt(t, filepath.Join(base, "first"))
	second := repoAt(t, filepath.Join(base, "second"))

	if _, err := state.Init(first, home); err != nil {
		t.Fatalf("first init: %v", err)
	}
	original, err := os.ReadFile(state.ConfigPath(home))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if _, err := state.Init(second, home); err != nil {
		t.Fatalf("second init: %v", err)
	}

	backup, err := os.ReadFile(state.ConfigPath(home) + ".petkit-backup")
	if err != nil {
		t.Fatalf("no backup of the replaced config: %v", err)
	}
	if string(backup) != string(original) {
		t.Errorf("the backup is %q, want %q", backup, original)
	}
}
