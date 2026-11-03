package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// Capture reads a machine's live ~/.claude.json and reduces its user-scope
// MCP servers to what settings/mcp.json records: name, transport, command,
// args, URL, and the NAMES of every env var and header — never a value.
//
// ⚠️ The reduction is the whole reason this exists, the same way PLUG-004 is
// the whole reason plugins.Capture exists. A live entry's `env` and `headers`
// maps hold secrets — API keys, bearer tokens — typed in when the server was
// added, and copying either map into the repository publishes them to
// everybody the repository is shared with. liveServer's Env and Headers
// fields are read exactly once, here, to list their key names; the values
// never leave this function.
//
// It is not a merge, for the same reason Capture (plugins) is not one: what
// this machine has replaces what the file said, because the file's whole
// claim is "this is what a machine has" — see Changes for what the
// difference is reported as.
func Capture(claudeJSONPath string) (*Desired, error) {
	live, err := LoadLive(claudeJSONPath)
	if err != nil {
		return nil, err
	}

	captured := &Desired{Servers: []Server{}}
	for name, entry := range live.Servers {
		server := Server{
			Name:      name,
			Transport: entry.Type,
			Command:   entry.Command,
			Args:      entry.Args,
			URL:       entry.URL,
		}
		if len(entry.Env) > 0 {
			server.EnvVars = sortedKeys(entry.Env)
		}
		if len(entry.Headers) > 0 {
			server.Headers = sortedKeys(entry.Headers)
		}
		captured.Servers = append(captured.Servers, server)
	}
	sort.Slice(captured.Servers, func(i, j int) bool {
		return captured.Servers[i].Name < captured.Servers[j].Name
	})

	return captured, nil
}

// Encode is the exact bytes settings/mcp.json is written with: two-space
// indentation and a trailing newline, with the servers in name order — the
// same spirit as plugins.Encode. Two captures of an unchanged machine produce
// the same bytes, which is what lets the command say "nothing changed" and
// write nothing.
func Encode(desired *Desired) ([]byte, error) {
	content, err := json.MarshalIndent(desired, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot encode the capture: %w", err)
	}
	return append(content, '\n'), nil
}

// Changes describes what a capture would do to the file, one line per
// difference, in a stable order. A capture that prints nothing changed
// nothing.
//
// before may be nil: a repository with no settings/mcp.json yet reports every
// server as an addition.
func Changes(before, after *Desired) []string {
	var lines []string

	oldServers := map[string]Server{}
	if before != nil {
		for _, server := range before.Servers {
			oldServers[server.Name] = server
		}
	}
	newServers := map[string]Server{}
	for _, server := range after.Servers {
		newServers[server.Name] = server
	}

	for _, server := range after.Servers {
		old, had := oldServers[server.Name]
		switch {
		case !had:
			lines = append(lines, fmt.Sprintf("+ server       %s (%s)", server.Name, server.Transport))
		case !reflect.DeepEqual(old, server):
			lines = append(lines, fmt.Sprintf("~ server       %s changed", server.Name))
		}
	}
	for _, name := range sortedKeys(oldServers) {
		if _, still := newServers[name]; !still {
			lines = append(lines, fmt.Sprintf("- server       %s (this machine no longer has it)", name))
		}
	}

	return lines
}
