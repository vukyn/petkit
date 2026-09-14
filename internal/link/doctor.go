package link

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
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

// Doctor runs the checks that are not about one item's link status: a source
// that is not there, a target outside the home directory, and whatever else is
// living in the skills directory.
func Doctor(m *manifest.Manifest, layout manifest.Layout) []Finding {
	var findings []Finding

	managed := newManagedTargets(len(m.Items), layout.GOOS)
	for _, item := range m.Items {
		source := item.SourcePath(m.Root, layout.GOOS)
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
		managed.claim(item, layout)
	}

	findings = append(findings, surveySkills(layout.SkillsDir(), managed, layout)...)
	return findings
}

// managedTargets is the set of paths the manifest claims. It answers "is this
// petkit's?" for every entry the survey turns up.
//
// ⚠️ It is keyed through ospath.Key rather than through the path itself,
// because on Windows the name read out of a directory and the target built from
// the manifest can differ in case — or in which separator the caller used — and
// still be one file. A miss here reports an item petkit manages as a stranger it
// does not, which is the opposite of what the survey is for.
type managedTargets struct {
	owners map[string]string
	goos   string
}

func newManagedTargets(size int, goos string) managedTargets {
	return managedTargets{owners: make(map[string]string, size), goos: goos}
}

func (t managedTargets) claim(item manifest.Item, layout manifest.Layout) {
	t.owners[t.key(item.TargetPath(layout))] = item.ID
}

// owner names the item that claims path, if one does.
func (t managedTargets) owner(path string) (string, bool) {
	id, found := t.owners[t.key(path)]
	return id, found
}

func (t managedTargets) key(path string) string {
	return ospath.Key(filepath.Clean(path), t.goos)
}

func surveySkills(skillsDir string, managed managedTargets, layout manifest.Layout) []Finding {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{
			Level:   Info,
			Message: fmt.Sprintf("cannot read %s: %v", layout.Display(skillsDir), err),
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
		shown := layout.Display(path)
		if broken, destination := brokenSymlink(path); broken {
			findings = append(findings, Finding{
				Level: Info,
				Message: fmt.Sprintf(
					"%s is a broken symlink: it points at %s, which does not exist", shown, destination),
			})
			continue
		}
		if _, isManaged := managed.owner(path); !isManaged {
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
