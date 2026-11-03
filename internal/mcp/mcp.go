// Package mcp compares the captured set of user-scope MCP servers against
// what a machine has configured and prints the commands that would close the
// gap.
//
// It never runs them, for the same reason package plugins never runs
// `claude plugin install`: `claude mcp add` already resolves the transport,
// writes into the machine's own config, and is the `claude` CLI's job.
// petkit's job is to say exactly what is missing, in a form that can be
// pasted.
//
// ⚠️ Scope boundary, decided and not reopened here (see TODO.md MCP-003):
// only USER-scope servers (`claude mcp add -s user ...`) are tracked.
// Project-scope servers live in the owning project's own `.mcp.json` and
// travel with that repository's clone, the same reasoning MFST-003 already
// applies to another repository's agents and commands. Local-scope servers
// are tied to one machine AND one project path at once, which a single
// cross-machine record in this repository cannot represent honestly — there
// is no "which machine, which checkout" field to put the other half in.
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vukyn/petkit/internal/ospath"
)

// placeholderValue stands in for an env var or header value that the live
// machine has and the manifest never stores — see the package doc on Server.
const placeholderValue = "<value: fill in>"

// Desired is settings/mcp.json: the captured user-scope MCP server set.
type Desired struct {
	Servers []Server `json:"servers"`
}

// Server is one user-scope MCP server the manifest records.
//
// ⚠️ EnvVars and Headers carry NAMES only, sorted, never the values a live
// machine's ~/.claude.json holds — that is the security-critical reduction
// PLUG-004 already set the pattern for, applied to a file this repository has
// no business publishing the contents of. See MCP-002.
//
// ⚠️ Command is kept as the absolute path a live machine recorded, unlike
// PLUG-004's installPath: it is necessary payload — `claude mcp add` cannot
// start a stdio server without it — so Build attaches an advisory Note
// instead of stripping it (see Build).
//
// ⚠️ There is no Version field. Measured 2026-11-03 against
// `claude mcp add --help`: no version-pinning flag exists for an MCP server,
// unlike PLUG-002's "reading, not a pin" — here there is nothing to read at
// all, so the field is absent rather than present-and-empty.
type Server struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	URL       string   `json:"url,omitempty"`
	EnvVars   []string `json:"envVars,omitempty"`
	Headers   []string `json:"headers,omitempty"`
}

// Live is what ~/.claude.json's top-level `mcpServers` key says about this
// machine.
type Live struct {
	Servers map[string]liveServer
}

// liveServer mirrors one entry of ~/.claude.json's `mcpServers` map, as much
// of it as petkit reads.
//
// ⚠️ Measured 2026-11-03 with `claude mcp add -s user petkit-probe -e
// PROBE_TOKEN=throwaway -- true` (stdio) and `claude mcp add -s user
// petkit-probe-http --transport http -H "Authorization: Bearer throwaway" --
// https://example.invalid/mcp` (http), diffing ~/.claude.json before and
// after each, then `claude mcp remove <name> -s user`. The measured shapes:
//
//	"petkit-probe":      {"type": "stdio", "command": "true", "args": [], "env": {"PROBE_TOKEN": "throwaway"}}
//	"petkit-probe-http": {"type": "http", "url": "https://example.invalid/mcp", "headers": {"Authorization": "Bearer throwaway"}}
//
// There is no `scope` field inside the entry at all — user scope IS the
// top-level `mcpServers` key, as opposed to the nested
// `projects["<path>"].mcpServers` a project- or local-scope add writes into
// instead. That is also how Capture knows these entries are user-scoped
// without reading a field that says so.
//
// Env and Headers are read once, inside Capture, purely to list their key
// names; nothing else in this package copies a value out of either map.
type liveServer struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	URL     string            `json:"url"`
	Env     map[string]string `json:"env"`
	Headers map[string]string `json:"headers"`
}

// Command is one line to run by hand, and the reason it is there.
type Command struct {
	Line string
	Why  string
}

// Plan is the printable result: commands to run, and notes that need no
// command.
type Plan struct {
	Commands []Command
	Notes    []string
}

// LoadDesired reads settings/mcp.json from the repository.
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

// LoadLive reads a machine's ~/.claude.json and returns its user-scope MCP
// servers. A missing file means a machine that has never configured one,
// which is a normal starting point and not an error — the same reading
// plugins.LoadLive gives a machine with no plugin files.
func LoadLive(claudeJSONPath string) (*Live, error) {
	live := &Live{Servers: map[string]liveServer{}}

	var document struct {
		McpServers map[string]liveServer `json:"mcpServers"`
	}
	content, err := os.ReadFile(claudeJSONPath)
	if os.IsNotExist(err) {
		return live, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", claudeJSONPath, err)
	}
	if err := json.Unmarshal(content, &document); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", claudeJSONPath, err)
	}
	if document.McpServers != nil {
		live.Servers = document.McpServers
	}
	return live, nil
}

// Build compares the two and produces the plan, in a stable order.
func Build(desired *Desired, live *Live) Plan {
	var plan Plan

	servers := make([]Server, len(desired.Servers))
	copy(servers, desired.Servers)
	sort.Slice(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })

	for _, server := range servers {
		if _, known := live.Servers[server.Name]; !known {
			plan.Commands = append(plan.Commands, Command{
				Line: server.addCommand(),
				Why: fmt.Sprintf(
					"not configured on this machine; env/header values are never recorded, so %s marks where to type one in",
					placeholderValue),
			})
		}
		// A server present on both sides is left alone: there is no version
		// concept to compare it against (see the Server doc), so a match is
		// simply nothing to report — the same as a matching plugin.
	}

	for _, server := range servers {
		if server.Command != "" && ospath.IsAbsAnywhere(server.Command) {
			plan.Notes = append(plan.Notes, fmt.Sprintf(
				"%s's command %q is an absolute path recorded on the machine that captured it; "+
					"a different machine may need a different one",
				server.Name, server.Command))
		}
	}

	extras := make([]string, 0)
	captured := map[string]bool{}
	for _, server := range desired.Servers {
		captured[server.Name] = true
	}
	for _, name := range sortedKeys(live.Servers) {
		if !captured[name] {
			extras = append(extras, name)
		}
	}
	for _, name := range extras {
		plan.Notes = append(plan.Notes, fmt.Sprintf(
			"%s is configured here but not in the manifest; petkit will not remove it", name))
	}

	return plan
}

// addCommand is the `claude mcp add -s user ...` line that would configure
// this server, with every value the manifest does not store shown as
// placeholderValue rather than left blank — a blank would read as "empty is
// fine", and placeholderValue reads as "fill this in by hand".
func (s Server) addCommand() string {
	parts := []string{"claude", "mcp", "add", "-s", "user"}
	if s.Transport != "" && s.Transport != "stdio" {
		parts = append(parts, "--transport", s.Transport)
	}
	parts = append(parts, s.Name)
	for _, name := range s.EnvVars {
		parts = append(parts, "-e", quoteForShell(name+"="+placeholderValue))
	}
	for _, name := range s.Headers {
		parts = append(parts, "-H", quoteForShell(name+": "+placeholderValue))
	}
	if s.Transport == "http" || s.Transport == "sse" {
		parts = append(parts, quoteForShell(s.URL))
		return strings.Join(parts, " ")
	}
	parts = append(parts, "--", quoteForShell(s.Command))
	for _, arg := range s.Args {
		parts = append(parts, quoteForShell(arg))
	}
	return strings.Join(parts, " ")
}

// quoteForShell is the single-quote wrap plugins.Build never needed: a
// plugin id cannot contain a space or a shell metacharacter, but an MCP
// server's command, its args, its URL, and the placeholder text standing in
// for an env var or header value all can.
func quoteForShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sortedKeys names in order, so printing a map never depends on Go's
// randomised map iteration.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
