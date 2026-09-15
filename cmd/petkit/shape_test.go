package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// CLI-011. ⚠️ Every other assertion in this package is strings.Contains over the
// whole buffer, which answers "does this phrase appear somewhere" — a much
// weaker question than the one the tests are named for. `CLI-010` added a new
// FIRST line to four commands and not one of 141 tests moved. The symmetric case
// is the one that will bite: a Contains cannot see a line disappear either, as
// long as its phrase survives somewhere else in the output.
//
// This file owns the other half: the **shape** of what each command prints —
// which lines, in what order, how many. The wording inside a line stays sampled,
// because output nobody can improve without a red suite is output that stops
// improving; `CLI-003` argues the same for messages that are the interface.
//
// The pattern language is one character: `…` matches any run. Everything else in
// a pattern is literal, each pattern is anchored to a whole line, and the line
// count must match exactly — that last part is what sees a line arrive or leave.

func TestEveryCommandPrintsTheShapeItIsMeantTo(t *testing.T) {
	resolved := `repository … (walked up from the working directory)`

	cases := []struct {
		name  string
		setup func(*sandbox)
		args  []string
		shape []string
	}{
		{
			name:  "status, with nothing installed yet",
			args:  []string{"status"},
			shape: []string{resolved, `missing…skill/writing-todo…`, `settings…`},
		},
		{
			name:  "sync, with nothing installed yet",
			args:  []string{"sync"},
			shape: []string{resolved, `create…skill/writing-todo…`, `1 change(s)`},
		},
		{
			name:  "sync, a second time",
			setup: func(s *sandbox) { s.run("sync") },
			args:  []string{"sync"},
			shape: []string{resolved, `ok…skill/writing-todo…`, `nothing to do; every item is already linked`},
		},
		{
			// ⚠️ A dry run with something to do prints NO closing line: the count
			// is suppressed and nothing replaces it, so the `would ` prefix on
			// each line is the whole of what says this did not happen. The case
			// below is the asymmetry — with nothing to do, a dry run does print a
			// closing line. Recorded rather than changed: `CLI-014`.
			name:  "sync --dry-run, with something to do",
			args:  []string{"sync", "--dry-run"},
			shape: []string{resolved, `would create…skill/writing-todo…`},
		},
		{
			name:  "sync --dry-run, with nothing to do",
			setup: func(s *sandbox) { s.run("sync") },
			args:  []string{"sync", "--dry-run"},
			shape: []string{resolved, `ok…skill/writing-todo…`, `nothing to do; every item is already linked`},
		},
		{
			name:  "doctor, with nothing to say",
			args:  []string{"doctor"},
			shape: []string{resolved, `nothing to report`},
		},
		{
			name: "doctor, with a stranger in the skills directory",
			setup: func(s *sandbox) {
				write(s.t, filepath.Join(s.home, ".claude", "skills", "stranger", "SKILL.md"), "not mine")
			},
			args:  []string{"doctor"},
			shape: []string{resolved, `info…stranger…`, ``, `0 problem(s), 1 note(s).…`},
		},
		{
			name:  "version",
			args:  []string{"version"},
			shape: []string{`petkit…`, `…items in…`},
		},
		{
			// ⚠️ setup is the one shape worth reading as a whole. Its last four
			// lines exist to say that it made no links — the property `README.md`
			// opens with — and a Contains suite could lose any of them without
			// noticing, including the blank lines that make it readable.
			name: "setup",
			args: []string{"setup", "…clone…"},
			shape: []string{
				`cloned…into…`,
				`recorded…in ~/.config/petkit/config.json`,
				``,
				`missing…skill/writing-todo…`,
				``,
				`setup made no links of its own. To install them:`,
				``,
				`  petkit sync`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newSandbox(t)
			if c.setup != nil {
				c.setup(s)
			}
			// ⚠️ `…clone…` stands for a path inside the sandbox. A relative
			// argument would resolve against the process's working directory,
			// which is this repository — a test that writes into the tree it is
			// measuring.
			args := append([]string(nil), c.args...)
			for i, arg := range args {
				if arg == "…clone…" {
					args[i] = filepath.Join(s.base, "clone")
				}
			}
			if code := s.run(args...); code != 0 {
				t.Fatalf("`petkit %s` exited %d: %s", strings.Join(c.args, " "), code, s.output())
			}
			assertShape(t, "petkit "+strings.Join(c.args, " "), s.stdout.String(), c.shape)
		})
	}
}

// ⚠️ assertShape decides whether the table above passes, so it owes its own
// tests: a helper nobody has watched fail is not evidence, it is a line somebody
// believes in. These are the two failures `CLI-011` exists for — a line arriving
// and a line leaving — and the last case is the one a strings.Contains suite
// cannot see at all.
func TestAssertShapeSeesALineArriveAndALineLeave(t *testing.T) {
	shape := []string{`repository …`, `ok…writing-todo…`, `nothing to do`}

	cases := []struct {
		name string
		out  string
		want bool
	}{
		{name: "exactly the shape", out: "repository /x\nok  writing-todo  /y\nnothing to do\n", want: true},
		{name: "a line arrived", out: "repository /x\nok  writing-todo  /y\nsomething new\nnothing to do\n"},
		{name: "a line left", out: "repository /x\nnothing to do\n"},
		{name: "the lines swapped", out: "ok  writing-todo  /y\nrepository /x\nnothing to do\n"},
		// ⚠️ The case this file exists for. `strings.Contains(out, "nothing to
		// do")` passes on this buffer; the phrase is there. The line it was
		// supposed to be is not, and an anchored pattern is what notices.
		{name: "the phrase survives in the wrong line", out: "repository /x\nok  writing-todo  /y\nthere is nothing to do here\n"},
	}
	for _, c := range cases {
		spy := &testing.T{}
		assertShape(spy, c.name, c.out, shape)
		if spy.Failed() == c.want {
			t.Errorf("%s: assertShape failed = %v, want %v", c.name, spy.Failed(), !c.want)
		}
	}
}

// The wording inside a line is sampled, not pinned: a message may be reworded
// around the parts a pattern names without turning this file red.
func TestAShapeDoesNotPinTheWordingAroundIt(t *testing.T) {
	shape := []string{`settings…`}
	for _, out := range []string{
		"settings   ~/.claude/settings.json is in step with the fragment\n",
		"settings   ~/.claude/settings.json has drifted from the fragment in 3 key(s)\n",
		"settings\n",
	} {
		spy := &testing.T{}
		assertShape(spy, "wording", out, shape)
		if spy.Failed() {
			t.Errorf("a reworded line turned the shape red: %q", out)
		}
	}
}

// assertShape reports whether out is exactly the lines shape describes, in
// order. A pattern matches one whole line, with `…` standing for any run.
func assertShape(t *testing.T, label, out string, shape []string) {
	t.Helper()
	lines := outputLines(out)
	if len(lines) != len(shape) {
		t.Errorf("%s printed %d line(s), want %d:\n%s", label, len(lines), len(shape), out)
		return
	}
	for i, pattern := range shape {
		if !lineMatches(lines[i], pattern) {
			t.Errorf("%s, line %d:\n  printed %q\n  want    %q", label, i+1, lines[i], pattern)
		}
	}
}

// outputLines is the lines of a command's output, with the trailing newline
// dropped so a well-formed output does not read as one line too many.
func outputLines(out string) []string {
	trimmed := strings.TrimSuffix(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func lineMatches(line, pattern string) bool {
	parts := strings.Split(pattern, "…")
	for i, part := range parts {
		parts[i] = regexp.QuoteMeta(part)
	}
	return regexp.MustCompile("^" + strings.Join(parts, ".*") + "$").MatchString(line)
}
