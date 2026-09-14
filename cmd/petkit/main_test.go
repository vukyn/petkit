package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
	"github.com/vukyn/petkit/internal/state"
)

// sandbox is a repository and a home directory built out of t.TempDir(). The
// home directory reaches the commands as a field, never through $HOME, so these
// tests cannot touch the machine they run on.
type sandbox struct {
	t      *testing.T
	base   string
	root   string
	home   string
	stdout *bytes.Buffer
	stderr *bytes.Buffer

	// env is the environment the commands read. Empty by default: a test that
	// wants CLAUDE_CONFIG_DIR or PETKIT_HOME seen puts it in here.
	env map[string]string

	// clones is what the fake git was asked to do, in order. Nothing in these
	// tests reaches the network: `clone` writes the same repository the sandbox
	// has, at whatever path it was given.
	clones [][]string
}

func newSandbox(t *testing.T) *sandbox {
	t.Helper()
	base := t.TempDir()
	s := &sandbox{
		t:      t,
		base:   base,
		root:   filepath.Join(base, "petkit"),
		home:   filepath.Join(base, "home"),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		env:    map[string]string{},
	}
	writeRepository(t, s.root)
	if err := os.MkdirAll(s.home, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return s
}

// writeRepository is a petkit checkout with one item in it.
func writeRepository(t *testing.T, root string) {
	t.Helper()
	write(t, filepath.Join(root, "skills", "writing-todo", "SKILL.md"), "from the repository")
	write(t, filepath.Join(root, "petkit.yaml"), `version: 1
items:
  - id: skill/writing-todo
    kind: skill
    source: skills/writing-todo
    target: ~/.claude/skills/writing-todo
`)
}

// fakeGit records what it was asked and answers `clone` by writing a checkout.
func (s *sandbox) fakeGit(t *testing.T) state.Runner {
	t.Helper()
	return func(_ string, args ...string) (string, error) {
		s.clones = append(s.clones, args)
		if len(args) == 3 && args[0] == "clone" {
			writeRepository(t, args[2])
		}
		return "", nil
	}
}

// layout is the sandbox's home directory described to the commands: this
// platform's path rules, and no CLAUDE_CONFIG_DIR unless a test sets one.
func (s *sandbox) layout() manifest.Layout {
	return manifest.NewLayout(s.home, ospath.Current(), s.environ)
}

// environ is what the commands read the environment through. It answers "" for
// everything until a test puts something in s.env.
func (s *sandbox) environ(name string) string { return s.env[name] }

func (s *sandbox) run(args ...string) int {
	s.stdout.Reset()
	s.stderr.Reset()
	return run(args, environment{
		layout:     s.layout(),
		workingDir: s.root,
		env:        s.environ,
		stdout:     s.stdout,
		stderr:     s.stderr,
		git:        s.fakeGit(s.t),
		modulePath: "example.com/someone/petkit",
	})
}

func (s *sandbox) output() string { return s.stdout.String() + s.stderr.String() }

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// status reports and does not judge: whatever it finds, the exit code is 0, so
// it can be used in a prompt or a script without guarding it.
func TestStatusExitsZeroEvenWhenItFindsAConflict(t *testing.T) {
	s := newSandbox(t)
	write(t, filepath.Join(s.home, ".claude", "skills", "writing-todo", "SKILL.md"), "somebody else's")

	if code := s.run("status"); code != 0 {
		t.Errorf("status exited %d, want 0: %s", code, s.output())
	}
	if !strings.Contains(s.stdout.String(), "conflict") {
		t.Errorf("status did not report the conflict: %s", s.stdout.String())
	}
}

// A conflict makes sync exit non-zero, so a script that syncs after a pull
// notices. With nothing in the way the identical command exits 0.
func TestSyncExitsNonZeroOnAConflictAndZeroWithoutOne(t *testing.T) {
	s := newSandbox(t)
	target := filepath.Join(s.home, ".claude", "skills", "writing-todo")
	write(t, filepath.Join(target, "SKILL.md"), "somebody else's")

	if code := s.run("sync"); code == 0 {
		t.Errorf("sync exited 0 with a conflict: %s", s.output())
	}
	if !strings.Contains(s.stderr.String(), "skill/writing-todo") {
		t.Errorf("the refusal does not name the item: %s", s.stderr.String())
	}
	if content, err := os.ReadFile(filepath.Join(target, "SKILL.md")); err != nil || string(content) != "somebody else's" {
		t.Errorf("the conflicting file did not survive: %q %v", content, err)
	}

	// The failing case: move the obstacle aside and the same command succeeds.
	if err := os.RemoveAll(target); err != nil {
		t.Fatalf("clear the target: %v", err)
	}
	if code := s.run("sync"); code != 0 {
		t.Errorf("sync exited %d with nothing in the way: %s", code, s.output())
	}
	if code := s.run("sync"); code != 0 {
		t.Errorf("a second sync exited %d: %s", code, s.output())
	}
	if !strings.Contains(s.stdout.String(), "nothing to do") {
		t.Errorf("the second sync did not report an unchanged machine: %s", s.stdout.String())
	}
}

// --dry-run reports the same refusal and the same exit code as the real run,
// so the preview cannot disagree with what happens.
func TestDryRunAgreesWithTheRealRunAboutConflicts(t *testing.T) {
	s := newSandbox(t)
	write(t, filepath.Join(s.home, ".claude", "skills", "writing-todo", "SKILL.md"), "somebody else's")

	code := s.run("sync", "--dry-run")
	if code == 0 {
		t.Errorf("a dry run over a conflict exited 0: %s", s.output())
	}
	if !strings.Contains(s.stdout.String(), "conflict") {
		t.Errorf("the dry run did not name the conflict: %s", s.stdout.String())
	}
	if strings.Contains(s.stdout.String(), "change(s)") {
		t.Errorf("the dry run claimed to have changed something: %s", s.stdout.String())
	}
}

// doctor exits non-zero only for a problem; the notes it prints about things
// other tools own do not fail the command.
func TestDoctorExitsZeroForNotesAndNonZeroForProblems(t *testing.T) {
	s := newSandbox(t)
	broken := filepath.Join(s.home, ".claude", "skills", "orchestration")
	if err := os.MkdirAll(filepath.Dir(broken), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink(filepath.Join(s.home, ".agents", "skills", "orchestration"), broken); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if code := s.run("doctor"); code != 0 {
		t.Errorf("doctor exited %d over a note: %s", code, s.output())
	}
	if !strings.Contains(s.stdout.String(), "broken symlink") {
		t.Errorf("doctor did not report the broken symlink: %s", s.stdout.String())
	}

	// The failing case: remove the source the manifest names, and the same
	// command has a problem to report.
	if err := os.RemoveAll(filepath.Join(s.root, "skills", "writing-todo")); err != nil {
		t.Fatalf("remove the source: %v", err)
	}
	if code := s.run("doctor"); code == 0 {
		t.Errorf("doctor exited 0 with a missing source: %s", s.output())
	}
}

// A manifest petkit refuses stops the command before anything is touched, and
// the message names the item.
func TestABadManifestStopsTheCommandAndNamesTheItem(t *testing.T) {
	s := newSandbox(t)
	write(t, filepath.Join(s.root, "petkit.yaml"), `version: 1
items:
  - id: skill/writing-todo
    source: skills/writing-todo
    target: /absolute/skills/writing-todo
`)
	if code := s.run("sync"); code == 0 {
		t.Fatalf("sync ran with an invalid manifest: %s", s.output())
	}
	if !strings.Contains(s.stderr.String(), "skill/writing-todo") {
		t.Errorf("the error does not name the item: %s", s.stderr.String())
	}
	if _, err := os.Stat(filepath.Join(s.home, ".claude")); !os.IsNotExist(err) {
		t.Error("the refused sync created something anyway")
	}
}

// The version is knowable even where the repository is not, so it is a note
// rather than a failure — a binary installed with `go install` and run from
// anywhere still answers.
func TestVersionAnswersWithoutARepository(t *testing.T) {
	s := newSandbox(t)
	elsewhere := t.TempDir()

	code := run([]string{"version"}, environment{
		layout:     s.layout(),
		workingDir: elsewhere,
		env:        s.environ,
		stdout:     s.stdout,
		stderr:     s.stderr,
	})
	if code != 0 {
		t.Errorf("version exited %d away from the repository: %s", code, s.output())
	}
	if !strings.Contains(s.stdout.String(), "petkit ") {
		t.Errorf("version printed no version: %s", s.stdout.String())
	}
	if !strings.Contains(s.stdout.String(), "petkit init") {
		t.Errorf("version did not say how to find the repository: %s", s.stdout.String())
	}

	// The failing case: inside the repository it counts the items instead.
	if code := s.run("version"); code != 0 {
		t.Fatalf("version exited %d inside the repository", code)
	}
	if !strings.Contains(s.stdout.String(), "1 items") {
		t.Errorf("version did not count the items: %s", s.stdout.String())
	}
}

// An unknown command is a usage error (2), which is not the same as a command
// that ran and failed (1).
func TestAnUnknownCommandIsAUsageError(t *testing.T) {
	s := newSandbox(t)
	if code := s.run("instal"); code != 2 {
		t.Errorf("an unknown command exited %d, want 2: %s", code, s.output())
	}
	if !strings.Contains(s.stderr.String(), "instal") {
		t.Errorf("the error does not quote what was typed: %s", s.stderr.String())
	}
	if !strings.Contains(s.stderr.String(), "unknown command") {
		t.Errorf("the error does not say what is wrong: %s", s.stderr.String())
	}
	if code := s.run(); code != 2 {
		t.Errorf("no command at all exited %d, want 2", code)
	}
	if code := s.run("help"); code != 0 {
		t.Errorf("help exited %d, want 0", code)
	}
}

// init records the repository, and afterwards the other commands work from a
// directory that has no petkit.yaml above it.
func TestInitLetsTheOtherCommandsRunFromAnywhere(t *testing.T) {
	s := newSandbox(t)
	elsewhere := t.TempDir()

	away := func(args ...string) int {
		s.stdout.Reset()
		s.stderr.Reset()
		return run(args, environment{
			layout:     s.layout(),
			workingDir: elsewhere,
			env:        s.environ,
			stdout:     s.stdout,
			stderr:     s.stderr,
		})
	}

	if code := away("status"); code == 0 {
		t.Fatalf("status found a repository before init: %s", s.output())
	}
	if code := away("init", s.root); code != 0 {
		t.Fatalf("init exited %d: %s", code, s.output())
	}
	if code := away("status"); code != 0 {
		t.Errorf("status exited %d after init: %s", code, s.output())
	}
	if !strings.Contains(s.stdout.String(), "skill/writing-todo") {
		t.Errorf("status did not read the recorded repository: %s", s.stdout.String())
	}
}

// settings apply through the CLI leaves the backup the rule promises.
func TestSettingsApplyThroughTheCommandLineLeavesABackup(t *testing.T) {
	s := newSandbox(t)
	write(t, filepath.Join(s.root, "settings", "fragment.json"), `{"model": "opus[1m]"}`)
	live := filepath.Join(s.home, ".claude", "settings.json")
	write(t, live, "{\n  \"model\": \"sonnet\",\n  \"keep\": {\"me\": 1}\n}\n")

	if code := s.run("settings", "diff"); code != 0 {
		t.Fatalf("settings diff exited %d: %s", code, s.output())
	}
	if content, _ := os.ReadFile(live); !strings.Contains(string(content), `"sonnet"`) {
		t.Error("settings diff changed the file")
	}

	if code := s.run("settings", "apply"); code != 0 {
		t.Fatalf("settings apply exited %d: %s", code, s.output())
	}
	content, err := os.ReadFile(live)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(content), "opus[1m]") {
		t.Errorf("the fragment was not applied: %s", content)
	}
	if !strings.Contains(string(content), `"keep"`) {
		t.Errorf("an undeclared key was dropped: %s", content)
	}

	entries, err := os.ReadDir(filepath.Dir(live))
	if err != nil {
		t.Fatalf("read the directory: %v", err)
	}
	backups := 0
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".petkit-backup-") {
			backups++
		}
	}
	if backups != 1 {
		t.Errorf("found %d backups, want 1", backups)
	}
}

// plugins plan prints commands and runs none: the machine it describes is not
// changed by describing it.
func TestPluginsPlanPrintsAndChangesNothing(t *testing.T) {
	s := newSandbox(t)
	write(t, filepath.Join(s.root, "settings", "plugins.json"), `{
      "marketplaces": {"context-mode": {"repo": "mksglu/context-mode", "source": "github"}},
      "plugins": [{"id": "context-mode@context-mode", "scope": "user", "version": "1.0.169"}],
      "enabled": {"context-mode@context-mode": true}
    }`)

	if code := s.run("plugins", "plan"); code != 0 {
		t.Fatalf("plugins plan exited %d: %s", code, s.output())
	}
	printed := s.stdout.String()
	for _, want := range []string{
		"claude plugin marketplace add mksglu/context-mode",
		"claude plugin install context-mode@context-mode",
		"ran none of them",
	} {
		if !strings.Contains(printed, want) {
			t.Errorf("the plan does not contain %q:\n%s", want, printed)
		}
	}
	if _, err := os.Stat(filepath.Join(s.home, ".claude", "plugins")); !os.IsNotExist(err) {
		t.Error("plugins plan created something under the home directory")
	}
}

// The help is the whole command surface, so it is asserted name by name. The
// help text is rendered from the command graph rather than written out beside
// it, which is what makes this test able to fail: a command dropped from the
// graph leaves the help, and this list stops matching.
func TestHelpListsEveryCommand(t *testing.T) {
	commands := []string{
		"setup", "status", "sync", "doctor", "settings", "plugins", "check", "version", "init",
	}

	// All three spellings reach the same help, and all three are worth 0.
	for _, invocation := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		s := newSandbox(t)
		if code := s.run(invocation...); code != 0 {
			t.Fatalf("%v exited %d, want 0: %s", invocation, code, s.output())
		}
		// Only the indented command rows count. The prose underneath the list
		// mentions `petkit init` too, and matching that would let a dropped
		// command pass.
		listed := map[string]bool{}
		for _, line := range strings.Split(s.stdout.String(), "\n") {
			name, ok := strings.CutPrefix(line, "  petkit ")
			if !ok {
				continue
			}
			if fields := strings.Fields(name); len(fields) > 0 {
				listed[fields[0]] = true
			}
		}
		for _, command := range commands {
			if !listed[command] {
				t.Errorf("%v does not list %q as a command:\n%s", invocation, command, s.stdout.String())
			}
		}
	}
}

// --dry-run is the flag most likely to be broken by moving the parsing into a
// framework, and the dry run itself is the promise that the preview costs
// nothing. Both are asserted here, at the command line, where the wiring is.
func TestSyncDryRunParsesAndWritesNothing(t *testing.T) {
	s := newSandbox(t)
	target := filepath.Join(s.home, ".claude", "skills", "writing-todo")

	if code := s.run("sync", "--dry-run"); code != 0 {
		t.Fatalf("sync --dry-run exited %d: %s", code, s.output())
	}
	if !strings.Contains(s.stdout.String(), "would create") {
		t.Errorf("the flag did not reach the plan — no preview was printed: %s", s.stdout.String())
	}
	if strings.Contains(s.stdout.String(), "change(s)") {
		t.Errorf("the dry run claimed to have changed something: %s", s.stdout.String())
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("the dry run created %s", target)
	}
	if _, err := os.Stat(filepath.Join(s.home, ".claude")); !os.IsNotExist(err) {
		t.Error("the dry run created something under the home directory")
	}

	// The positive case the refusal owes: without the flag the identical
	// command does make the link, so the test above is measuring the flag and
	// not a sync that never works.
	if code := s.run("sync"); code != 0 {
		t.Fatalf("sync exited %d: %s", code, s.output())
	}
	link, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("sync left no symlink at %s: %v", target, err)
	}
	if link != filepath.Join(s.root, "skills", "writing-todo") {
		t.Errorf("the link points at %s, want the repository's copy", link)
	}
}

// `petkit version` is the authority on what a version looks like, so the
// framework's --version flag has to print what it prints — not the one-line
// "petkit version X" the framework would print left to itself.
func TestTheVersionFlagPrintsWhatTheVersionCommandPrints(t *testing.T) {
	s := newSandbox(t)

	if code := s.run("version"); code != 0 {
		t.Fatalf("version exited %d: %s", code, s.output())
	}
	fromCommand := s.stdout.String()

	if code := s.run("--version"); code != 0 {
		t.Fatalf("--version exited %d: %s", code, s.output())
	}
	if fromFlag := s.stdout.String(); fromFlag != fromCommand {
		t.Errorf("--version printed\n%q\nbut version printed\n%q", fromFlag, fromCommand)
	}
	if !strings.Contains(fromCommand, "1 items") {
		t.Errorf("neither spelling counted the items: %s", fromCommand)
	}
}

// A command line that cannot be parsed is a usage error (2) like an unknown
// command, not a command that ran and failed (1). Every one of these was a
// hand-written branch before the framework took the parsing over.
func TestTheUsageErrorsAreAllWorthTwo(t *testing.T) {
	for _, invocation := range [][]string{
		{"sync", "--no-such-flag"},
		{"settings"},
		{"settings", "nope"},
		{"plugins"},
		{"plugins", "nope"},
	} {
		s := newSandbox(t)
		if code := s.run(invocation...); code != 2 {
			t.Errorf("%v exited %d, want 2: %s", invocation, code, s.output())
		}
		if s.stderr.Len() == 0 {
			t.Errorf("%v failed without saying why", invocation)
		}
	}

	// The positive case each refusal owes: the spellings these reject are
	// rejected because they are wrong, not because the command is broken.
	for _, invocation := range [][]string{
		{"sync", "--dry-run"},
		{"settings", "diff"},
		{"plugins", "plan"},
	} {
		s := newSandbox(t)
		write(t, filepath.Join(s.root, "settings", "fragment.json"), `{"model": "opus[1m]"}`)
		write(t, filepath.Join(s.root, "settings", "plugins.json"), `{"marketplaces": {}, "plugins": []}`)
		if code := s.run(invocation...); code != 0 {
			t.Errorf("%v exited %d, want 0: %s", invocation, code, s.output())
		}
	}
}

// homeTree is every path under the home directory with what kind of thing it
// is. Symlinks are reported as symlinks rather than followed, because whether
// one appeared is the whole question setup has to answer.
func homeTree(t *testing.T, home string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		relative, err := filepath.Rel(home, path)
		if err != nil {
			return err
		}
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			tree[relative] = "symlink"
		case entry.IsDir():
			tree[relative] = "directory"
		default:
			tree[relative] = "file"
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", home, err)
	}
	return tree
}

// setup gets a machine a clone and a record, and stops there. ⚠️ The links are
// sync's to make: the home directory is snapshotted either side of the command
// and the only thing that may appear in it is the record itself.
func TestSetupWithoutSyncMakesNoLinks(t *testing.T) {
	s := newSandbox(t)
	target := filepath.Join(s.base, "clone")
	before := homeTree(t, s.home)

	if code := s.run("setup", target); code != 0 {
		t.Fatalf("setup exited %d: %s", code, s.output())
	}

	after := homeTree(t, s.home)
	for path, kind := range after {
		if kind == "symlink" {
			t.Errorf("setup created a symlink at %s", path)
		}
		if _, existed := before[path]; existed {
			continue
		}
		if !strings.HasPrefix(path, ".config") {
			t.Errorf("setup created %s (%s), which is not the record", path, kind)
		}
	}
	if _, err := os.Stat(filepath.Join(s.home, ".claude")); !os.IsNotExist(err) {
		t.Error("setup created something under ~/.claude")
	}

	// What it does instead: report what sync would do, and name the command.
	printed := s.stdout.String()
	for _, want := range []string{
		"https://example.com/someone/petkit",
		"missing",
		"skill/writing-todo",
		"petkit sync",
	} {
		if !strings.Contains(printed, want) {
			t.Errorf("setup did not print %q:\n%s", want, printed)
		}
	}
}

// --sync goes all the way, and carries sync's exit code with it: a conflict is
// still refused, and the file in the way is still there afterwards.
func TestSetupWithSyncInstallsTheLinksAndKeepsSyncsExitCode(t *testing.T) {
	s := newSandbox(t)
	target := filepath.Join(s.base, "clone")
	installed := filepath.Join(s.home, ".claude", "skills", "writing-todo")

	if code := s.run("setup", "--sync", target); code != 0 {
		t.Fatalf("setup --sync exited %d: %s", code, s.output())
	}
	link, err := os.Readlink(installed)
	if err != nil {
		t.Fatalf("setup --sync left no symlink at %s: %v", installed, err)
	}
	if want := filepath.Join(target, "skills", "writing-todo"); link != want {
		t.Errorf("the link points at %s, want %s", link, want)
	}

	// The refusal: a fresh machine with something real already sitting at the
	// target gets the conflict and the non-zero exit, through setup exactly as
	// through sync.
	other := newSandbox(t)
	obstacle := filepath.Join(other.home, ".claude", "skills", "writing-todo")
	write(t, filepath.Join(obstacle, "SKILL.md"), "somebody else's")

	if code := other.run("setup", "--sync", filepath.Join(other.base, "clone")); code == 0 {
		t.Errorf("setup --sync exited 0 over a conflict: %s", other.output())
	}
	if !strings.Contains(other.stderr.String(), "skill/writing-todo") {
		t.Errorf("the refusal does not name the item: %s", other.stderr.String())
	}
	if content, err := os.ReadFile(filepath.Join(obstacle, "SKILL.md")); err != nil || string(content) != "somebody else's" {
		t.Errorf("the conflicting file did not survive: %q %v", content, err)
	}
}

// A machine that already has a repository is told where it is rather than given
// a second one, and the message carries the recorded path.
func TestSetupRefusesAMachineThatIsAlreadySetUp(t *testing.T) {
	s := newSandbox(t)
	if code := s.run("init", s.root); code != 0 {
		t.Fatalf("init exited %d: %s", code, s.output())
	}

	if code := s.run("setup"); code == 0 {
		t.Fatalf("setup ran on a machine that already had a repository: %s", s.output())
	}
	if !strings.Contains(s.stderr.String(), s.root) {
		t.Errorf("the refusal does not name the recorded repository: %s", s.stderr.String())
	}
	if len(s.clones) != 0 {
		t.Errorf("git ran anyway: %v", s.clones)
	}

	// The positive case: naming somewhere else is an explicit second clone.
	if code := s.run("setup", filepath.Join(s.base, "second")); code != 0 {
		t.Errorf("setup refused an explicit second path: %s", s.output())
	}
}

// Claude Code honours CLAUDE_CONFIG_DIR and petkit hard-coded ~/.claude, so on
// a machine that sets it every link was installed into a directory nothing
// reads: the tool reported success and the skills were never seen.
//
// ⚠️ This is a defect on every platform, not a Windows one. It is measured end
// to end — through `sync` and `doctor`, not through the resolver — because the
// hard-coded path was in the commands rather than in the manifest logic.
func TestSyncInstallsIntoClaudeConfigDir(t *testing.T) {
	s := newSandbox(t)
	configDir := filepath.Join(s.base, "elsewhere-config")
	s.env[manifest.EnvConfigDir] = configDir

	if code := s.run("sync"); code != 0 {
		t.Fatalf("sync exited %d: %s", code, s.output())
	}

	installed := filepath.Join(configDir, "skills", "writing-todo")
	destination, err := os.Readlink(installed)
	if err != nil {
		t.Fatalf("nothing was installed at %s: %v", installed, err)
	}
	if want := filepath.Join(s.root, "skills", "writing-todo"); destination != want {
		t.Errorf("the link points at %s, want %s", destination, want)
	}
	// ⚠️ And nothing was left in the directory the variable said not to use.
	if _, err := os.Stat(filepath.Join(s.home, ".claude")); !os.IsNotExist(err) {
		t.Error("sync also wrote into ~/.claude, which this machine does not read")
	}

	// doctor surveys the same directory, or it reports a managed item as an
	// unmanaged stranger and an installed skill as missing.
	if code := s.run("doctor"); code != 0 {
		t.Fatalf("doctor exited %d: %s", code, s.output())
	}
	if strings.Contains(s.output(), "is not in the manifest") {
		t.Errorf("doctor did not recognise the item it just installed: %s", s.output())
	}
	if strings.Contains(s.output(), "which is not there") {
		t.Errorf("doctor reported a problem with a machine that has none: %s", s.output())
	}
}

// The positive sibling: with the variable unset the links land in ~/.claude
// exactly as they always did, so the variable adds a case rather than moving
// the default.
func TestWithoutClaudeConfigDirSyncStillInstallsIntoDotClaude(t *testing.T) {
	s := newSandbox(t)

	if code := s.run("sync"); code != 0 {
		t.Fatalf("sync exited %d: %s", code, s.output())
	}
	if _, err := os.Readlink(filepath.Join(s.home, ".claude", "skills", "writing-todo")); err != nil {
		t.Errorf("nothing was installed in ~/.claude: %v", err)
	}
}

// The settings merge and the plugin survey read out of the same directory, so
// they follow the variable too — a machine that moved its configuration should
// not have half of petkit looking in the old place.
func TestSettingsAndPluginsFollowClaudeConfigDir(t *testing.T) {
	s := newSandbox(t)
	configDir := filepath.Join(s.base, "elsewhere-config")
	s.env[manifest.EnvConfigDir] = configDir
	write(t, filepath.Join(s.root, "settings", "fragment.json"), `{"model": "opus"}`)
	write(t, filepath.Join(configDir, "settings.json"), `{"model": "sonnet", "keepMe": 1}`)

	if code := s.run("settings", "apply"); code != 0 {
		t.Fatalf("settings apply exited %d: %s", code, s.output())
	}
	merged, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if err != nil {
		t.Fatalf("read the merged settings: %v", err)
	}
	if !strings.Contains(string(merged), `"opus"`) {
		t.Errorf("the fragment was not merged into the configured directory: %s", merged)
	}
	if !strings.Contains(string(merged), "keepMe") {
		t.Errorf("an unrelated key did not survive the merge: %s", merged)
	}
	if _, err := os.Stat(filepath.Join(s.home, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Error("settings apply wrote into ~/.claude as well")
	}
}

// doctor surveys the skills directory for entries petkit does not manage, and
// that directory moves with CLAUDE_CONFIG_DIR too. A doctor still looking at
// ~/.claude finds an empty or absent directory and reports nothing at all —
// which reads exactly like a clean machine.
func TestDoctorSurveysTheClaudeConfigDir(t *testing.T) {
	s := newSandbox(t)
	configDir := filepath.Join(s.base, "elsewhere-config")
	s.env[manifest.EnvConfigDir] = configDir

	if code := s.run("sync"); code != 0 {
		t.Fatalf("sync exited %d: %s", code, s.output())
	}
	write(t, filepath.Join(configDir, "skills", "stranger", "SKILL.md"), "somebody else's")

	if code := s.run("doctor"); code != 0 {
		t.Fatalf("doctor exited %d: %s", code, s.output())
	}
	if !strings.Contains(s.output(), "stranger") {
		t.Errorf("doctor did not survey the configured directory at all: %s", s.output())
	}
	if !strings.Contains(s.output(), "stranger is not in the manifest") {
		t.Errorf("doctor did not report the stranger it found: %s", s.output())
	}
	// The item petkit installed a moment ago is its own, not a stranger.
	if strings.Contains(s.output(), "writing-todo is not in the manifest") {
		t.Errorf("doctor called its own item unmanaged: %s", s.output())
	}
}
