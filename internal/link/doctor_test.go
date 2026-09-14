package link_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/link"
)

func findingsMentioning(findings []link.Finding, needle string) []link.Finding {
	var matched []link.Finding
	for _, finding := range findings {
		if strings.Contains(finding.Message, needle) {
			matched = append(matched, finding)
		}
	}
	return matched
}

// The real case from the author's machine: ~/.claude/skills/orchestration is a
// symlink into ~/.agents/skills and its destination does not exist. doctor says
// so, and says it as information — the link is not petkit's to repair.
func TestDoctorReportsABrokenSymlinkUnderTheSkillsDirectory(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	loaded := m.manifestWith(skillItem("writing-todo"))

	destination := filepath.Join(m.home, ".agents", "skills", "orchestration")
	broken := filepath.Join(m.home, ".claude", "skills", "orchestration")
	mustSymlink(t, destination, broken)

	if _, err := os.Stat(broken); !os.IsNotExist(err) {
		t.Fatalf("the fixture is not a broken symlink: stat gave %v", err)
	}

	findings := link.Doctor(loaded, m.layout)
	matched := findingsMentioning(findings, "orchestration")
	if len(matched) != 1 {
		t.Fatalf("doctor produced %d findings about orchestration, want 1: %v", len(matched), findings)
	}
	if !strings.Contains(matched[0].Message, "broken symlink") {
		t.Errorf("message = %q, want it to say the symlink is broken", matched[0].Message)
	}
	if !strings.Contains(matched[0].Message, destination) {
		t.Errorf("message = %q, want it to name %s", matched[0].Message, destination)
	}
	if matched[0].Level != link.Info {
		t.Errorf("level = %q, want %q: a broken link someone else owns is information", matched[0].Level, link.Info)
	}
}

// The failing case: a symlink whose destination exists and which the manifest
// knows about produces no finding at all, so the report above is a real signal.
func TestDoctorIsSilentAboutAWorkingManagedLink(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	loaded := m.manifestWith(skillItem("writing-todo"))

	mustSymlink(t, filepath.Join(m.root, "skills", "writing-todo"), m.targetOf("writing-todo"))

	findings := link.Doctor(loaded, m.layout)
	if len(findings) != 0 {
		t.Fatalf("doctor reported %v, want nothing", findings)
	}
}

// Anything under the skills directory that the manifest does not name is
// reported as information: most of it is owned by a plugin.
func TestDoctorReportsAnUnmanagedSkill(t *testing.T) {
	m := newMachine(t)
	m.skill("writing-todo", "one")
	loaded := m.manifestWith(skillItem("writing-todo"))

	mustWrite(t, filepath.Join(m.home, ".claude", "skills", "brainstorming", "SKILL.md"), "from a plugin")

	findings := link.Doctor(loaded, m.layout)
	matched := findingsMentioning(findings, "brainstorming")
	if len(matched) != 1 {
		t.Fatalf("doctor produced %d findings about brainstorming, want 1: %v", len(matched), findings)
	}
	if matched[0].Level != link.Info {
		t.Errorf("level = %q, want %q", matched[0].Level, link.Info)
	}
	if !strings.Contains(matched[0].Message, "leaves it alone") {
		t.Errorf("message = %q, want it to say petkit will not touch it", matched[0].Message)
	}
}

// A source named by the manifest but absent from the repository is a problem,
// because it is the one thing on this list petkit's own repository controls.
func TestDoctorReportsAManifestSourceThatIsNotThere(t *testing.T) {
	m := newMachine(t)
	loaded := m.manifestWith(skillItem("writing-todo")) // never created on disk

	findings := link.Doctor(loaded, m.layout)
	matched := findingsMentioning(findings, "skill/writing-todo")
	if len(matched) != 1 {
		t.Fatalf("doctor produced %d findings about the missing source, want 1: %v", len(matched), findings)
	}
	if matched[0].Level != link.Problem {
		t.Errorf("level = %q, want %q", matched[0].Level, link.Problem)
	}

	// The failing case: create the source and the problem goes away.
	m.skill("writing-todo", "one")
	if remaining := findingsMentioning(link.Doctor(loaded, m.layout), "skill/writing-todo"); len(remaining) != 0 {
		t.Errorf("the finding survived the fix: %v", remaining)
	}
}
