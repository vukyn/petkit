package link

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vukyn/petkit/internal/manifest"
)

// Level separates "you have to fix this" from "you should know this". Most of
// what doctor finds around ~/.claude/skills is legitimately owned elsewhere —
// by a plugin, or by another tool — so it is reported and not judged.
type Level string

const (
	// Problem: the manifest or the machine is wrong and petkit cannot do its job.
	Problem Level = "problem"
	// Info: worth knowing, owned by someone else, not petkit's to change.
	Info Level = "info"
)

// Finding is one thing doctor noticed.
type Finding struct {
	Level   Level
	Message string
}

// SkillsDirName is the directory doctor surveys for entries petkit does not manage.
const SkillsDirName = ".claude/skills"

// Doctor runs the checks that are not about one item's link status: a source
// that is not there, a target outside the home directory, and whatever else is
// living under ~/.claude/skills.
func Doctor(m *manifest.Manifest, home string) []Finding {
	var findings []Finding

	managed := make(map[string]string, len(m.Items))
	for _, item := range m.Items {
		source := item.SourcePath(m.Root)
		if _, err := os.Stat(source); err != nil {
			findings = append(findings, Finding{
				Level: Problem,
				Message: fmt.Sprintf(
					"item %q names source %s, which is not there; add it to the repository or remove the item",
					item.ID, source),
			})
		}

		// ⚠️ No check here that the target stays under the home directory.
		// manifest.Validate refuses `~/..` outright, so a manifest that reaches
		// this function cannot carry an escaping target — a branch here could not
		// fire, and a guard that cannot fire is not a guard.
		managed[filepath.Clean(item.TargetPath(home))] = item.ID
	}

	findings = append(findings, surveySkills(filepath.Join(home, filepath.FromSlash(SkillsDirName)), managed, home)...)
	return findings
}

func surveySkills(skillsDir string, managed map[string]string, home string) []Finding {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{
			Level:   Info,
			Message: fmt.Sprintf("cannot read %s: %v", manifest.CollapseHome(skillsDir, home), err),
		}}
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	var findings []Finding
	for _, name := range names {
		path := filepath.Join(skillsDir, name)
		shown := manifest.CollapseHome(path, home)
		if broken, destination := brokenSymlink(path); broken {
			findings = append(findings, Finding{
				Level: Info,
				Message: fmt.Sprintf(
					"%s is a broken symlink: it points at %s, which does not exist", shown, destination),
			})
			continue
		}
		if _, isManaged := managed[filepath.Clean(path)]; !isManaged {
			findings = append(findings, Finding{
				Level: Info,
				Message: fmt.Sprintf(
					"%s is not in the manifest; petkit leaves it alone", shown),
			})
		}
	}
	return findings
}

// brokenSymlink reports a symlink whose destination is not there. Stat follows
// the link, so a symlink that Lstats but does not Stat is exactly this case.
func brokenSymlink(path string) (bool, string) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false, ""
	}
	if _, err := os.Stat(path); err == nil {
		return false, ""
	}
	destination, err := os.Readlink(path)
	if err != nil {
		return true, "an unreadable destination"
	}
	return true, destination
}
