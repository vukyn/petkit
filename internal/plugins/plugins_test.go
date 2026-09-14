package plugins_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/plugins"
)

func desired() *plugins.Desired {
	return &plugins.Desired{
		Marketplaces: map[string]plugins.Marketplace{
			"context-mode": {Repo: "mksglu/context-mode", Source: "github"},
			"caveman":      {Repo: "JuliusBrussee/caveman", Source: "github"},
		},
		Plugins: []plugins.Plugin{
			{ID: "context-mode@context-mode", Scope: "user", Version: "1.0.169"},
			{ID: "caveman@caveman", Scope: "user", Version: "15581d14007f"},
		},
		Enabled: map[string]bool{"context-mode@context-mode": true},
	}
}

func lines(plan plugins.Plan) []string {
	var out []string
	for _, command := range plan.Commands {
		out = append(out, command.Line)
	}
	return out
}

// An empty machine gets one command per marketplace and one per plugin, and
// the enable that the manifest records.
func TestAnEmptyMachineGetsEveryMarketplaceAndPlugin(t *testing.T) {
	plan := plugins.Build(desired(), &plugins.Live{
		Marketplaces: map[string]string{},
		Installed:    map[string]string{},
		Enabled:      map[string]bool{},
	})

	want := []string{
		"claude plugin marketplace add JuliusBrussee/caveman",
		"claude plugin marketplace add mksglu/context-mode",
		"claude plugin install context-mode@context-mode",
		"claude plugin install caveman@caveman",
		"claude plugin enable context-mode@context-mode",
	}
	got := lines(plan)
	if len(got) != len(want) {
		t.Fatalf("plan has %d commands, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("command %d is %q, want %q", index, got[index], want[index])
		}
	}
}

// The failing case: a machine that already matches gets no commands at all.
func TestAMatchingMachineGetsNoCommands(t *testing.T) {
	plan := plugins.Build(desired(), &plugins.Live{
		Marketplaces: map[string]string{
			"context-mode": "mksglu/context-mode",
			"caveman":      "JuliusBrussee/caveman",
		},
		Installed: map[string]string{
			"context-mode@context-mode": "1.0.169",
			"caveman@caveman":           "15581d14007f",
		},
		Enabled: map[string]bool{"context-mode@context-mode": true},
	})
	if len(plan.Commands) != 0 {
		t.Errorf("a matching machine produced %v", lines(plan))
	}
	if len(plan.Notes) != 0 {
		t.Errorf("a matching machine produced notes: %v", plan.Notes)
	}
}

// A different installed version is a note, not a command: petkit does not guess
// at how somebody wants to resolve it.
func TestADifferentVersionIsANoteAndNotACommand(t *testing.T) {
	plan := plugins.Build(desired(), &plugins.Live{
		Marketplaces: map[string]string{
			"context-mode": "mksglu/context-mode",
			"caveman":      "JuliusBrussee/caveman",
		},
		Installed: map[string]string{
			"context-mode@context-mode": "1.0.100",
			"caveman@caveman":           "15581d14007f",
		},
		Enabled: map[string]bool{"context-mode@context-mode": true},
	})
	if len(plan.Commands) != 0 {
		t.Errorf("a version difference produced commands: %v", lines(plan))
	}
	if len(plan.Notes) != 1 || !strings.Contains(plan.Notes[0], "1.0.100") {
		t.Errorf("notes = %v, want one naming the installed version", plan.Notes)
	}
}

// A plugin the manifest never captured is left alone and only mentioned.
func TestAnExtraInstalledPluginIsOnlyMentioned(t *testing.T) {
	plan := plugins.Build(desired(), &plugins.Live{
		Marketplaces: map[string]string{
			"context-mode": "mksglu/context-mode",
			"caveman":      "JuliusBrussee/caveman",
		},
		Installed: map[string]string{
			"context-mode@context-mode": "1.0.169",
			"caveman@caveman":           "15581d14007f",
			"something-else@elsewhere":  "2.0.0",
		},
		Enabled: map[string]bool{"context-mode@context-mode": true},
	})
	if len(plan.Commands) != 0 {
		t.Errorf("an extra plugin produced commands: %v", lines(plan))
	}
	if len(plan.Notes) != 1 || !strings.Contains(plan.Notes[0], "will not remove it") {
		t.Errorf("notes = %v, want one saying petkit leaves it alone", plan.Notes)
	}
}

// A machine that has never installed a plugin has no files to read, and that
// is a normal starting point rather than an error.
func TestLoadLiveAcceptsAMachineWithNoPluginFiles(t *testing.T) {
	live, err := plugins.LoadLive(filepath.Join(t.TempDir(), "plugins"))
	if err != nil {
		t.Fatalf("a machine with no plugin files failed: %v", err)
	}
	if len(live.Installed) != 0 || len(live.Marketplaces) != 0 {
		t.Errorf("an empty machine reported %v and %v", live.Marketplaces, live.Installed)
	}
}

// The shapes of the two files on a real machine, read as they are written.
func TestLoadLiveReadsTheRealFileShapes(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "known_marketplaces.json"), `{
      "caveman": {
        "source": {"source": "github", "repo": "JuliusBrussee/caveman"},
        "installLocation": "/Users/someone/.claude/plugins/marketplaces/caveman",
        "lastUpdated": "2026-09-13T00:00:00.000Z"
      }
    }`)
	write(t, filepath.Join(dir, "installed_plugins.json"), `{
      "version": 1,
      "plugins": {
        "caveman@caveman": [{"scope": "user", "installPath": "/somewhere", "version": "15581d14007f"}]
      },
      "enabledPlugins": {"context-mode@context-mode": true}
    }`)

	live, err := plugins.LoadLive(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if live.Marketplaces["caveman"] != "JuliusBrussee/caveman" {
		t.Errorf("marketplace repo = %q", live.Marketplaces["caveman"])
	}
	if live.Installed["caveman@caveman"] != "15581d14007f" {
		t.Errorf("installed version = %q", live.Installed["caveman@caveman"])
	}
	if !live.Enabled["context-mode@context-mode"] {
		t.Error("the enabled plugin was not read")
	}
}

// The repository's own capture parses, and every plugin in it names a
// marketplace the same file describes.
func TestTheRepositoryCaptureIsSelfConsistent(t *testing.T) {
	path := filepath.Join("..", "..", "settings", "plugins.json")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no capture to check: %v", err)
	}
	loaded, err := plugins.LoadDesired(path)
	if err != nil {
		t.Fatalf("the repository's capture does not parse: %v", err)
	}
	for _, plugin := range loaded.Plugins {
		at := strings.LastIndex(plugin.ID, "@")
		if at < 0 {
			t.Errorf("plugin id %q has no marketplace", plugin.ID)
			continue
		}
		marketplace := plugin.ID[at+1:]
		if _, known := loaded.Marketplaces[marketplace]; !known {
			t.Errorf("plugin %q names marketplace %q, which the capture does not describe",
				plugin.ID, marketplace)
		}
	}
}

// liveMachine writes the two files a real machine keeps in ~/.claude/plugins,
// in the shape the real ones have — including every field the capture must not
// carry across. The paths in it are the thing under test, so they are named
// constants and asserted against by name.
const (
	liveProjectPath     = "/Users/someone/work/a-clients-private-repo"
	liveInstallPath     = "/Users/someone/.claude/plugins/cache/caveman"
	liveInstallLocation = "/Users/someone/.claude/plugins/marketplaces/caveman"
)

func liveMachine(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "known_marketplaces.json"), `{
      "caveman": {
        "source": {"source": "github", "repo": "JuliusBrussee/caveman"},
        "installLocation": "`+liveInstallLocation+`",
        "lastUpdated": "2026-09-13T00:00:00.000Z"
      },
      "context-mode": {
        "source": {"source": "github", "repo": "mksglu/context-mode"},
        "installLocation": "`+liveInstallLocation+`",
        "lastUpdated": "2026-09-13T00:00:00.000Z"
      }
    }`)
	write(t, filepath.Join(dir, "installed_plugins.json"), `{
      "version": 1,
      "plugins": {
        "caveman@caveman": [{
          "scope": "project",
          "projectPath": "`+liveProjectPath+`",
          "installPath": "`+liveInstallPath+`",
          "version": "15581d14007f",
          "installedAt": "2026-09-13T00:00:00.000Z",
          "lastUpdated": "2026-09-13T00:00:00.000Z",
          "gitCommitSha": "b36e0829aa11"
        }],
        "context-mode@context-mode": [{
          "scope": "user",
          "installPath": "`+liveInstallPath+`",
          "version": "1.0.169"
        }]
      },
      "enabledPlugins": {"context-mode@context-mode": true}
    }`)
	return dir
}

// ⚠️ This is the test PLUG-004 exists for. The obvious way to rebuild
// settings/plugins.json is to copy installed_plugins.json, which carries an
// absolute path into whatever repository each project-scoped plugin was
// installed for — a path nobody wants published, and a private repository is
// still shared with everybody who is added to it.
//
// The assertion searches the BYTES THAT WOULD BE WRITTEN, not the struct. A
// capture whose reduction can only be confirmed by reading the code is the same
// hand-reduction it replaces, with more steps.
func TestACaptureCarriesNoPathOutOfTheLiveFiles(t *testing.T) {
	captured, err := plugins.Capture(liveMachine(t))
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	content, err := plugins.Encode(captured)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	for _, leak := range []string{
		liveProjectPath, liveInstallPath, liveInstallLocation,
		"projectPath", "installPath", "installLocation",
		"installedAt", "lastUpdated", "gitCommitSha",
		"/Users/", "a-clients-private-repo",
	} {
		if strings.Contains(string(content), leak) {
			t.Errorf("the capture carries %q across:\n%s", leak, content)
		}
	}

	// The positive case the refusal owes: the three fields it is supposed to
	// keep, and the marketplace's source, are all there — so the test above is
	// measuring the reduction and not a capture that writes nothing.
	for _, want := range []string{
		`"id": "caveman@caveman"`,
		`"scope": "project"`,
		`"version": "15581d14007f"`,
		`"repo": "JuliusBrussee/caveman"`,
		`"source": "github"`,
		`"context-mode@context-mode": true`,
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("the capture does not contain %q:\n%s", want, content)
		}
	}
}

// The capture of an unchanged machine is the same bytes every time, and the
// same bytes after a round trip through the file. That is what lets the command
// say "nothing changed" and write nothing — without it, map ordering alone
// would rewrite the file on every run and the backup directory would fill up
// with identical copies.
func TestTwoCapturesOfOneMachineAreTheSameBytes(t *testing.T) {
	dir := liveMachine(t)

	first, err := plugins.Capture(dir)
	if err != nil {
		t.Fatalf("first capture: %v", err)
	}
	firstBytes, err := plugins.Encode(first)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Twenty, not two: the failure this is watching for is map iteration order
	// reaching the output, and map iteration order is random per range, so one
	// comparison would agree by chance half the time.
	for attempt := 0; attempt < 20; attempt++ {
		again, err := plugins.Capture(dir)
		if err != nil {
			t.Fatalf("capture %d: %v", attempt, err)
		}
		content, err := plugins.Encode(again)
		if err != nil {
			t.Fatalf("encode %d: %v", attempt, err)
		}
		if string(content) != string(firstBytes) {
			t.Fatalf("capture %d differs:\n%s\nwant\n%s", attempt, content, firstBytes)
		}
	}

	// And the file petkit writes reads back as the same capture: the encoding
	// is not lossy in the direction `plugins plan` reads it.
	path := filepath.Join(t.TempDir(), "plugins.json")
	write(t, path, string(firstBytes))
	reloaded, err := plugins.LoadDesired(path)
	if err != nil {
		t.Fatalf("the capture does not parse: %v", err)
	}
	roundTripped, err := plugins.Encode(reloaded)
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if string(roundTripped) != string(firstBytes) {
		t.Errorf("a round trip changed the file:\n%s\nwant\n%s", roundTripped, firstBytes)
	}
	if changes := plugins.Changes(reloaded, first); len(changes) != 0 {
		t.Errorf("a capture of an unchanged machine reports %v", changes)
	}
}

// What a capture reports is what moved, and nothing else.
func TestACaptureReportsWhatMoved(t *testing.T) {
	before := desired()
	after := &plugins.Desired{
		Marketplaces: map[string]plugins.Marketplace{
			"context-mode": {Repo: "mksglu/context-mode", Source: "github"},
			"newcomer":     {Repo: "someone/newcomer", Source: "github"},
		},
		Plugins: []plugins.Plugin{
			{ID: "context-mode@context-mode", Scope: "user", Version: "1.0.200"},
			{ID: "newcomer@newcomer", Scope: "user", Version: "0.1.0"},
		},
		Enabled: map[string]bool{"context-mode@context-mode": true},
	}

	got := strings.Join(plugins.Changes(before, after), "\n")
	for _, want := range []string{
		"+ marketplace  newcomer",
		"- marketplace  caveman",
		"~ plugin       context-mode@context-mode last seen 1.0.169 -> 1.0.200",
		"+ plugin       newcomer@newcomer",
		"- plugin       caveman@caveman",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not contain %q:\n%s", want, got)
		}
	}

	// The failing case: nothing moved, nothing is reported.
	if changes := plugins.Changes(before, desired()); len(changes) != 0 {
		t.Errorf("an unchanged capture reported %v", changes)
	}
}

// ⚠️ PLUG-002. `claude plugin install --help` offers --config, --json,
// -s/--scope and -y/--yes and NO version flag at all (measured 2026-09-14), so
// a version cannot be pinned through it and the recorded one is a reading. The
// test holds both halves of that: the command never carries a version, and
// every line that shows one says what it is.
func TestAVersionIsAReadingAndTheInstallCommandNeverPinsIt(t *testing.T) {
	empty := plugins.Build(desired(), &plugins.Live{
		Marketplaces: map[string]string{},
		Installed:    map[string]string{},
		Enabled:      map[string]bool{},
	})
	for _, command := range empty.Commands {
		if !strings.HasPrefix(command.Line, "claude plugin install ") {
			continue
		}
		if strings.Contains(command.Line, "1.0.169") || strings.Contains(command.Line, "15581d14007f") {
			t.Errorf("the plan pinned a version claude plugin install cannot take: %q", command.Line)
		}
		if !strings.Contains(command.Why, "last seen") {
			t.Errorf("the reason calls the version something other than a reading: %q", command.Why)
		}
	}

	// The other place a version is shown: a machine that has the plugin at a
	// different version.
	behind := plugins.Build(desired(), &plugins.Live{
		Marketplaces: map[string]string{
			"context-mode": "mksglu/context-mode",
			"caveman":      "JuliusBrussee/caveman",
		},
		Installed: map[string]string{
			"context-mode@context-mode": "1.0.100",
			"caveman@caveman":           "15581d14007f",
		},
		Enabled: map[string]bool{"context-mode@context-mode": true},
	})
	if len(behind.Notes) != 1 {
		t.Fatalf("notes = %v, want one", behind.Notes)
	}
	if !strings.Contains(behind.Notes[0], "last seen") {
		t.Errorf("the note calls the version something other than a reading: %q", behind.Notes[0])
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
