package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/mcp"
)

func desired() *mcp.Desired {
	return &mcp.Desired{
		Servers: []mcp.Server{
			{
				Name:      "secrets-server",
				Transport: "stdio",
				Command:   "/usr/local/bin/secrets-server",
				Args:      []string{"--flag", "value with spaces", "quo'te"},
				EnvVars:   []string{"API_KEY"},
			},
			{
				Name:      "remote-server",
				Transport: "http",
				URL:       "https://example.invalid/mcp",
				Headers:   []string{"Authorization"},
			},
		},
	}
}

func lines(plan mcp.Plan) []string {
	var out []string
	for _, command := range plan.Commands {
		out = append(out, command.Line)
	}
	return out
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// liveWithNoServers is a machine that has never configured a user-scope MCP
// server: there is no ~/.claude.json at all, which LoadLive treats as a
// normal starting point rather than an error.
func liveWithNoServers(t *testing.T) *mcp.Live {
	t.Helper()
	live, err := mcp.LoadLive(filepath.Join(t.TempDir(), ".claude.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return live
}

// An empty machine gets one add command per captured server, each quoted for
// the shell and showing the placeholder for every env var and header name the
// manifest does not store a value for.
func TestAnEmptyMachineGetsEveryServerAdded(t *testing.T) {
	plan := mcp.Build(desired(), liveWithNoServers(t))

	want := []string{
		"claude mcp add -s user --transport http remote-server -H 'Authorization: <value: fill in>' 'https://example.invalid/mcp'",
		`claude mcp add -s user secrets-server -e 'API_KEY=<value: fill in>' -- '/usr/local/bin/secrets-server' '--flag' 'value with spaces' 'quo'\''te'`,
	}
	got := lines(plan)
	if len(got) != len(want) {
		t.Fatalf("plan has %d commands, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("command %d is\n  %q\nwant\n  %q", index, got[index], want[index])
		}
	}
}

// Shell quoting is the one thing plugins.Build never needed: a plugin id
// cannot contain a space or a shell metacharacter, but an MCP arg can.
func TestTheAddCommandQuotesArgsContainingSpacesAndQuotes(t *testing.T) {
	plan := mcp.Build(desired(), liveWithNoServers(t))

	var secretsLine string
	for _, command := range plan.Commands {
		if strings.Contains(command.Line, "secrets-server") {
			secretsLine = command.Line
		}
	}
	if secretsLine == "" {
		t.Fatalf("no command for secrets-server: %v", lines(plan))
	}
	if !strings.Contains(secretsLine, `'value with spaces'`) {
		t.Errorf("the space-containing arg was not quoted: %s", secretsLine)
	}
	if !strings.Contains(secretsLine, `'quo'\''te'`) {
		t.Errorf("the embedded single quote was not escaped: %s", secretsLine)
	}
}

// The failing case: a machine that already has every server (under whatever
// values it has) gets no commands at all. There is no version concept to
// compare a present server against — see the Server doc — so a name match is
// simply nothing to report.
func TestAMatchingMachineGetsNoCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	write(t, path, `{
      "mcpServers": {
        "secrets-server": {
          "type": "stdio",
          "command": "/usr/local/bin/secrets-server",
          "args": ["--flag", "value with spaces", "quo'te"],
          "env": {"API_KEY": "whatever-this-machine-has"}
        },
        "remote-server": {
          "type": "http",
          "url": "https://example.invalid/mcp",
          "headers": {"Authorization": "whatever-this-machine-has"}
        }
      }
    }`)
	live, err := mcp.LoadLive(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	plan := mcp.Build(desired(), live)
	if len(plan.Commands) != 0 {
		t.Errorf("a matching machine produced %v", lines(plan))
	}
	for _, note := range plan.Notes {
		if !strings.Contains(note, "absolute path") {
			t.Errorf("a matching machine produced an unexpected note: %v", plan.Notes)
		}
	}
}

// A server the manifest never captured is left alone and only mentioned —
// never turned into a `claude mcp remove` command. Unlike plugins, which has
// no analogous temptation (there is no `claude plugin remove`-shaped escape
// hatch sitting right there), `claude mcp remove` exists and a naive
// implementation could reach for it.
func TestAnExtraOnMachineServerIsOnlyMentionedNeverRemoved(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	write(t, path, `{
      "mcpServers": {
        "secrets-server": {"type": "stdio", "command": "/usr/local/bin/secrets-server"},
        "remote-server": {"type": "http", "url": "https://example.invalid/mcp"},
        "extra-server": {"type": "stdio", "command": "/opt/extra"}
      }
    }`)
	live, err := mcp.LoadLive(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	plan := mcp.Build(desired(), live)
	if len(plan.Commands) != 0 {
		t.Errorf("an extra server produced commands: %v", lines(plan))
	}
	for _, command := range plan.Commands {
		if strings.Contains(command.Line, "remove") {
			t.Errorf("an extra server produced a remove command: %v", lines(plan))
		}
	}

	extraNotes := 0
	for _, note := range plan.Notes {
		if strings.Contains(note, "claude mcp remove") {
			t.Errorf("an extra server note suggested removing it: %q", note)
		}
		if strings.Contains(note, "extra-server") {
			extraNotes++
			if !strings.Contains(note, "will not remove it") {
				t.Errorf("the note does not say petkit leaves it alone: %q", note)
			}
		}
	}
	if extraNotes != 1 {
		t.Errorf("found %d notes mentioning extra-server, want 1: %v", extraNotes, plan.Notes)
	}
}

// The absolute command path desired() carries is necessary payload — `claude
// mcp add` cannot start a stdio server without it — so it earns an advisory
// note rather than the PLUG-004-style stripping a path with no run-time use
// gets.
func TestAnAbsoluteCommandPathProducesAnAdvisoryNote(t *testing.T) {
	plan := mcp.Build(desired(), liveWithNoServers(t))

	found := false
	for _, note := range plan.Notes {
		if strings.Contains(note, "secrets-server") &&
			strings.Contains(note, "/usr/local/bin/secrets-server") &&
			strings.Contains(note, "absolute path") {
			found = true
		}
	}
	if !found {
		t.Errorf("no advisory note for the absolute command path: %v", plan.Notes)
	}
}

// A machine that has never configured a server has no ~/.claude.json to read,
// and that is a normal starting point rather than an error.
func TestLoadLiveAcceptsAMachineWithNoClaudeJSON(t *testing.T) {
	live, err := mcp.LoadLive(filepath.Join(t.TempDir(), ".claude.json"))
	if err != nil {
		t.Fatalf("a machine with no .claude.json failed: %v", err)
	}
	if len(live.Servers) != 0 {
		t.Errorf("an empty machine reported %v", live.Servers)
	}
}

// The shapes of a real machine's ~/.claude.json mcpServers entries, read as
// measured. See mcp.go's liveServer doc for the exact commands and date.
func TestLoadLiveReadsTheRealFileShapes(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	write(t, path, `{
      "numStartups": 42,
      "mcpServers": {
        "petkit-probe": {
          "type": "stdio",
          "command": "true",
          "args": [],
          "env": {"PROBE_TOKEN": "throwaway"}
        },
        "petkit-probe-http": {
          "type": "http",
          "url": "https://example.invalid/mcp",
          "headers": {"Authorization": "Bearer throwaway"}
        }
      }
    }`)

	live, err := mcp.LoadLive(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	stdioEntry, ok := live.Servers["petkit-probe"]
	if !ok {
		t.Fatal("petkit-probe was not read")
	}
	if stdioEntry.Type != "stdio" || stdioEntry.Command != "true" {
		t.Errorf("stdio entry = %+v", stdioEntry)
	}
	if stdioEntry.Env["PROBE_TOKEN"] != "throwaway" {
		t.Errorf("stdio env = %v", stdioEntry.Env)
	}

	httpEntry, ok := live.Servers["petkit-probe-http"]
	if !ok {
		t.Fatal("petkit-probe-http was not read")
	}
	if httpEntry.Type != "http" || httpEntry.URL != "https://example.invalid/mcp" {
		t.Errorf("http entry = %+v", httpEntry)
	}
	if httpEntry.Headers["Authorization"] != "Bearer throwaway" {
		t.Errorf("http headers = %v", httpEntry.Headers)
	}
}

// liveMachine writes a ~/.claude.json with the fields a capture must not
// carry across: an env value and a header value, each a named constant so
// the leak this test watches for is never an inline literal somebody could
// mistake for a real secret.
const (
	liveEnvValue    = "env-value-should-never-be-written"
	liveHeaderValue = "header-value-should-never-be-written"
)

func liveMachine(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".claude.json")
	write(t, path, `{
      "mcpServers": {
        "secrets-server": {
          "type": "stdio",
          "command": "/usr/local/bin/secrets-server",
          "args": ["--flag", "value with spaces"],
          "env": {"API_KEY": "`+liveEnvValue+`"}
        },
        "remote-server": {
          "type": "http",
          "url": "https://example.invalid/mcp",
          "headers": {"Authorization": "`+liveHeaderValue+`"}
        }
      }
    }`)
	return path
}

// ⚠️ This is the test MCP-002 exists for. The obvious way to rebuild
// settings/mcp.json is to copy ~/.claude.json's mcpServers map, which carries
// the literal env var and header VALUES a server was configured with — a
// private API key or bearer token, typed in once and then published to
// everybody the repository is shared with.
//
// The assertion searches the BYTES THAT WOULD BE WRITTEN, not the struct —
// the same discipline PLUG-004's test holds plugins.Capture to.
func TestACaptureCarriesNoEnvOrHeaderValueAcrossOnlyNames(t *testing.T) {
	captured, err := mcp.Capture(liveMachine(t))
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	content, err := mcp.Encode(captured)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	for _, leak := range []string{liveEnvValue, liveHeaderValue} {
		if strings.Contains(string(content), leak) {
			t.Errorf("the capture carries %q across:\n%s", leak, content)
		}
	}

	// The positive case the refusal owes: the NAMES the reduction is supposed
	// to keep are there, so the test above is measuring the reduction and not
	// a capture that writes nothing.
	for _, want := range []string{
		`"API_KEY"`, `"Authorization"`,
		`"name": "secrets-server"`, `"name": "remote-server"`,
		`"command": "/usr/local/bin/secrets-server"`,
		`"url": "https://example.invalid/mcp"`,
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("the capture does not contain %q:\n%s", want, content)
		}
	}
}

// The capture of an unchanged machine is the same bytes every time, and the
// same bytes after a round trip through the file — the same property
// plugins.Capture is held to, and for the same reason: it is what lets the
// command say "nothing changed" and write nothing.
func TestTwoCapturesOfOneMachineAreTheSameBytes(t *testing.T) {
	path := liveMachine(t)

	first, err := mcp.Capture(path)
	if err != nil {
		t.Fatalf("first capture: %v", err)
	}
	firstBytes, err := mcp.Encode(first)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Twenty, not two: the failure this watches for is map iteration order
	// reaching the output, and that order is random per range.
	for attempt := range 20 {
		again, err := mcp.Capture(path)
		if err != nil {
			t.Fatalf("capture %d: %v", attempt, err)
		}
		content, err := mcp.Encode(again)
		if err != nil {
			t.Fatalf("encode %d: %v", attempt, err)
		}
		if string(content) != string(firstBytes) {
			t.Fatalf("capture %d differs:\n%s\nwant\n%s", attempt, content, firstBytes)
		}
	}

	writtenPath := filepath.Join(t.TempDir(), "mcp.json")
	write(t, writtenPath, string(firstBytes))
	reloaded, err := mcp.LoadDesired(writtenPath)
	if err != nil {
		t.Fatalf("the capture does not parse: %v", err)
	}
	roundTripped, err := mcp.Encode(reloaded)
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if string(roundTripped) != string(firstBytes) {
		t.Errorf("a round trip changed the file:\n%s\nwant\n%s", roundTripped, firstBytes)
	}
	if changes := mcp.Changes(reloaded, first); len(changes) != 0 {
		t.Errorf("a capture of an unchanged machine reports %v", changes)
	}
}

// What a capture reports is what moved, and nothing else.
func TestACaptureReportsWhatMoved(t *testing.T) {
	before := desired()
	after := &mcp.Desired{
		Servers: []mcp.Server{
			{
				Name: "remote-server", Transport: "http",
				URL: "https://example.invalid/mcp", Headers: []string{"Authorization"},
			},
			{Name: "newcomer", Transport: "stdio", Command: "/opt/newcomer"},
		},
	}

	got := strings.Join(mcp.Changes(before, after), "\n")
	for _, want := range []string{
		"+ server       newcomer (stdio)",
		"- server       secrets-server (this machine no longer has it)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the report does not contain %q:\n%s", want, got)
		}
	}

	// The failing case: nothing moved, nothing is reported.
	if changes := mcp.Changes(before, desired()); len(changes) != 0 {
		t.Errorf("an unchanged capture reported %v", changes)
	}
}
