package plugins

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Capture reads a machine's live plugin files and reduces them to what
// settings/plugins.json records: each marketplace's source, and {id, scope,
// version} per plugin.
//
// ⚠️ The reduction is the whole reason this exists. The live
// installed_plugins.json carries an `installPath` on every entry and a
// `projectPath` — an absolute path into whatever repository the plugin was
// installed for — on every project-scoped one, and known_marketplaces.json
// carries an `installLocation`. Copying either file into the repository
// publishes those paths to everybody the repository is shared with. The
// reduction is performed by the types: liveMarketplace and liveInstalled do not
// declare those fields, so they are never even read, and Desired has nowhere to
// put them if they were.
//
// It is not a merge. What this machine has replaces what the file said, because
// the file's whole claim is "this is what a machine has" — see Changes for what
// the difference is reported as.
func Capture(pluginsDir string) (*Desired, error) {
	known, installed, err := readLiveFiles(pluginsDir)
	if err != nil {
		return nil, err
	}

	captured := &Desired{
		Enabled:      map[string]bool{},
		Marketplaces: map[string]Marketplace{},
		Plugins:      []Plugin{},
	}

	for name, entry := range known {
		captured.Marketplaces[name] = Marketplace{
			Repo:   entry.Source.Repo,
			Source: entry.Source.Source,
		}
	}

	for id, entries := range installed.Plugins {
		if len(entries) == 0 {
			continue
		}
		// The first entry, which is the one the plan already reads. An id
		// installed at two scopes at once is a machine to look at by hand, not
		// a shape to guess at here.
		captured.Plugins = append(captured.Plugins, Plugin{
			ID:      id,
			Scope:   entries[0].Scope,
			Version: entries[0].Version,
		})
	}
	sort.Slice(captured.Plugins, func(i, j int) bool {
		return captured.Plugins[i].ID < captured.Plugins[j].ID
	})

	for id, enabled := range installed.EnabledPlugins {
		captured.Enabled[id] = enabled
	}

	return captured, nil
}

// Encode is the exact bytes settings/plugins.json is written with: two-space
// indentation and a trailing newline, with the plugins in id order and the maps
// in the key order encoding/json sorts them into. Two captures of an unchanged
// machine produce the same bytes, which is what lets the command say "nothing
// changed" and write nothing.
func Encode(desired *Desired) ([]byte, error) {
	content, err := json.MarshalIndent(desired, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot encode the capture: %w", err)
	}
	return append(content, '\n'), nil
}

// Changes describes what a capture would do to the file, one line per
// difference, in a stable order. A capture that prints nothing changed nothing.
//
// before may be nil: a repository with no settings/plugins.json yet reports
// every marketplace and plugin as an addition.
func Changes(before, after *Desired) []string {
	var lines []string

	oldMarketplaces := map[string]Marketplace{}
	oldPlugins := map[string]Plugin{}
	oldEnabled := map[string]bool{}
	if before != nil {
		oldMarketplaces = before.Marketplaces
		for _, plugin := range before.Plugins {
			oldPlugins[plugin.ID] = plugin
		}
		oldEnabled = before.Enabled
	}

	for _, name := range sortedKeys(after.Marketplaces) {
		marketplace := after.Marketplaces[name]
		switch old, had := oldMarketplaces[name]; {
		case !had:
			lines = append(lines, fmt.Sprintf("+ marketplace  %s (%s %s)",
				name, marketplace.Source, marketplace.Repo))
		case old != marketplace:
			lines = append(lines, fmt.Sprintf("~ marketplace  %s (%s %s -> %s %s)",
				name, old.Source, old.Repo, marketplace.Source, marketplace.Repo))
		}
	}
	for _, name := range sortedKeys(oldMarketplaces) {
		if _, still := after.Marketplaces[name]; !still {
			lines = append(lines, fmt.Sprintf("- marketplace  %s (this machine does not know it)", name))
		}
	}

	newPlugins := map[string]Plugin{}
	for _, plugin := range after.Plugins {
		newPlugins[plugin.ID] = plugin
	}
	for _, plugin := range after.Plugins {
		old, had := oldPlugins[plugin.ID]
		switch {
		case !had:
			lines = append(lines, fmt.Sprintf("+ plugin       %s (%s, last seen %s)",
				plugin.ID, plugin.Scope, plugin.Version))
		case old.Version != plugin.Version:
			lines = append(lines, fmt.Sprintf("~ plugin       %s last seen %s -> %s",
				plugin.ID, old.Version, plugin.Version))
		case old.Scope != plugin.Scope:
			lines = append(lines, fmt.Sprintf("~ plugin       %s scope %s -> %s",
				plugin.ID, old.Scope, plugin.Scope))
		}
	}
	for _, id := range sortedKeys(oldPlugins) {
		if _, still := newPlugins[id]; !still {
			lines = append(lines, fmt.Sprintf("- plugin       %s (not installed here)", id))
		}
	}

	for _, id := range sortedKeys(after.Enabled) {
		old, had := oldEnabled[id]
		switch {
		case !had:
			lines = append(lines, fmt.Sprintf("+ enabled      %s = %t", id, after.Enabled[id]))
		case old != after.Enabled[id]:
			lines = append(lines, fmt.Sprintf("~ enabled      %s %t -> %t", id, old, after.Enabled[id]))
		}
	}
	for _, id := range sortedKeys(oldEnabled) {
		if _, still := after.Enabled[id]; !still {
			lines = append(lines, fmt.Sprintf("- enabled      %s (this machine says nothing about it)", id))
		}
	}

	return lines
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
