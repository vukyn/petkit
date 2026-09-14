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

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
