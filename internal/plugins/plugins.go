// Package plugins compares the captured plugin set against what a machine has
// and prints the commands that would close the gap.
//
// It never runs them. Installing a plugin talks to the network, writes into
// ~/.claude/plugins and is the `claude` CLI's job; petkit's job is to say
// exactly what is missing, in a form that can be pasted.
package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Desired is settings/plugins.json: the captured plugin set.
//
// ⚠️ The field order is the order `petkit plugins capture` writes them in, and
// it is the order the committed file already has. Encoding is the only reason
// it matters; decoding does not care.
type Desired struct {
	Enabled      map[string]bool        `json:"enabled"`
	Marketplaces map[string]Marketplace `json:"marketplaces"`
	Plugins      []Plugin               `json:"plugins"`
}

// Marketplace is where a plugin comes from.
type Marketplace struct {
	Repo   string `json:"repo"`
	Source string `json:"source"`
}

// Plugin is one plugin the manifest records.
//
// ⚠️ Version is the version **last seen** on the machine that captured it, not
// a version to install. `claude plugin install` takes `--config`, `--json`,
// `-s/--scope` and `-y/--yes` and **no version flag at all** (measured
// 2026-09-14), so there is nothing to pin it to; at least one marketplace also
// carries `autoUpdate: true`, which would move it underneath a pin if one
// existed. It is kept because a recorded reading is how a machine notices it is
// running something older — see PLUG-002.
type Plugin struct {
	ID      string `json:"id"`
	Scope   string `json:"scope"`
	Version string `json:"version"`
}

// Live is what ~/.claude/plugins says about this machine.
type Live struct {
	Marketplaces map[string]string // name -> repo
	Installed    map[string]string // plugin id -> installed version
	Enabled      map[string]bool
}

// Command is one line to run by hand, and the reason it is there.
type Command struct {
	Line string
	Why  string
}

// Plan is the printable result: commands to run, and notes that need no command.
type Plan struct {
	Commands []Command
	Notes    []string
}

// LoadDesired reads settings/plugins.json from the repository.
func LoadDesired(path string) (*Desired, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var desired Desired
	if err := json.Unmarshal(content, &desired); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", path, err)
	}
	return &desired, nil
}

// LoadLive reads a machine's plugin directory. A missing file means a machine
// that has never installed anything, which is a normal starting point and not
// an error.
func LoadLive(pluginsDir string) (*Live, error) {
	live := &Live{
		Marketplaces: map[string]string{},
		Installed:    map[string]string{},
		Enabled:      map[string]bool{},
	}

	known, installed, err := readLiveFiles(pluginsDir)
	if err != nil {
		return nil, err
	}
	for name, entry := range known {
		live.Marketplaces[name] = entry.Source.Repo
	}
	for id, entries := range installed.Plugins {
		version := ""
		if len(entries) > 0 {
			version = entries[0].Version
		}
		live.Installed[id] = version
	}
	for id, enabled := range installed.EnabledPlugins {
		live.Enabled[id] = enabled
	}
	return live, nil
}

// liveMarketplace is one entry of ~/.claude/plugins/known_marketplaces.json, as
// much of it as petkit reads.
//
// ⚠️ The file's entries also carry `installLocation` — an absolute path into
// the machine's plugin cache — and `lastUpdated`. Neither is declared here, so
// neither can be read, and a capture built out of this type has nowhere to put
// them even by accident.
type liveMarketplace struct {
	Source struct {
		Source string `json:"source"`
		Repo   string `json:"repo"`
	} `json:"source"`
}

// liveInstalled is ~/.claude/plugins/installed_plugins.json, as much of it as
// petkit reads.
//
// ⚠️ Each entry in the real file also carries `installPath`, `installedAt`,
// `gitCommitSha` and — on every project-scoped entry — `projectPath`, an
// absolute path into whatever repository the plugin was installed for. None of
// them is declared here. That is the reduction PLUG-004 is about, and it is
// enforced by the type rather than by remembering to drop fields.
type liveInstalled struct {
	Plugins map[string][]struct {
		Scope   string `json:"scope"`
		Version string `json:"version"`
	} `json:"plugins"`
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// readLiveFiles reads both of a machine's plugin files. It is one function
// because both readers — the plan's and the capture's — must agree about what
// the files contain; two readers is two answers.
func readLiveFiles(pluginsDir string) (map[string]liveMarketplace, liveInstalled, error) {
	var known map[string]liveMarketplace
	var installed liveInstalled

	if err := readJSONIfPresent(filepath.Join(pluginsDir, "known_marketplaces.json"), &known); err != nil {
		return nil, installed, err
	}
	if err := readJSONIfPresent(filepath.Join(pluginsDir, "installed_plugins.json"), &installed); err != nil {
		return nil, installed, err
	}
	return known, installed, nil
}

func readJSONIfPresent(path string, into any) error {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	if err := json.Unmarshal(content, into); err != nil {
		return fmt.Errorf("cannot parse %s: %w", path, err)
	}
	return nil
}

// Build compares the two and produces the plan, in a stable order.
func Build(desired *Desired, live *Live) Plan {
	var plan Plan

	names := make([]string, 0, len(desired.Marketplaces))
	for name := range desired.Marketplaces {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		marketplace := desired.Marketplaces[name]
		reference := marketplace.Repo
		if reference == "" {
			reference = name
		}
		if _, known := live.Marketplaces[name]; !known {
			plan.Commands = append(plan.Commands, Command{
				Line: fmt.Sprintf("claude plugin marketplace add %s", reference),
				Why:  fmt.Sprintf("marketplace %q is not known to this machine", name),
			})
		}
	}

	for _, plugin := range desired.Plugins {
		installedVersion, installed := live.Installed[plugin.ID]
		if !installed {
			// ⚠️ The id, and nothing else. `claude plugin install` has no
			// version flag, so there is no version to append here — and a
			// recorded version that looked like an argument would be read as
			// one. See the Plugin type: the version is a reading.
			plan.Commands = append(plan.Commands, Command{
				Line: fmt.Sprintf("claude plugin install %s", plugin.ID),
				Why: fmt.Sprintf("not installed; last seen at %s (%s scope) — a reading, not a pin, "+
					"so this installs whatever the marketplace offers today",
					plugin.Version, plugin.Scope),
			})
			continue
		}
		if installedVersion != plugin.Version {
			plan.Notes = append(plan.Notes, fmt.Sprintf(
				"%s is installed at %s; last seen at %s — the manifest records a reading, not a pin, "+
					"so this is a difference to look at rather than one to correct",
				plugin.ID, installedVersion, plugin.Version))
		}
	}

	enabledIDs := make([]string, 0, len(desired.Enabled))
	for id := range desired.Enabled {
		enabledIDs = append(enabledIDs, id)
	}
	sort.Strings(enabledIDs)
	for _, id := range enabledIDs {
		want := desired.Enabled[id]
		if live.Enabled[id] == want {
			continue
		}
		verb := "enable"
		if !want {
			verb = "disable"
		}
		plan.Commands = append(plan.Commands, Command{
			Line: fmt.Sprintf("claude plugin %s %s", verb, id),
			Why:  fmt.Sprintf("the manifest has it %sd", verb),
		})
	}

	extras := make([]string, 0)
	captured := map[string]bool{}
	for _, plugin := range desired.Plugins {
		captured[plugin.ID] = true
	}
	for id := range live.Installed {
		if !captured[id] {
			extras = append(extras, id)
		}
	}
	sort.Strings(extras)
	for _, id := range extras {
		plan.Notes = append(plan.Notes, fmt.Sprintf(
			"%s is installed here but not in the manifest; petkit will not remove it", id))
	}

	return plan
}
