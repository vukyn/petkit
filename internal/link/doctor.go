package link

import (
	"errors"
	"fmt"
	"io/fs"
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
		switch state, destination, err := followSymlink(path, os.Stat); state {
		case dangling:
			findings = append(findings, Finding{
				Level: Info,
				Message: fmt.Sprintf(
					"%s is a broken symlink: it points at %s, which does not exist", shown, destination),
			})
			continue
		case unfollowable:
			findings = append(findings, Finding{
				Level: Info,
				Message: fmt.Sprintf(
					"%s is a symlink petkit cannot follow: it points at %s, and following it failed: %s",
					shown, destination, reason(err)),
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

// statFunc is os.Stat with the call as a parameter.
//
// ⚠️ It is a parameter for the same reason the platform is one elsewhere in this
// repository: the error Windows produces for a file-flavour symlink cannot be
// staged on a Unix filesystem, so it is handed to the decision instead. A branch
// nobody has watched fail is not evidence that it works. LINK-006.
type statFunc func(string) (os.FileInfo, error)

// symlinkState is what became of an entry when doctor tried to follow it.
type symlinkState int

const (
	// notASymlink: an ordinary file or directory, or nothing at all.
	notASymlink symlinkState = iota
	// followed: the destination is there.
	followed
	// dangling: the destination is not there.
	dangling
	// unfollowable: the destination could not be opened, and ⚠️ **that is not
	// the same fact as its being absent**. doctor printed "which does not
	// exist" for this case too, which on Windows was a false statement about
	// the filesystem, repeated on every run — LINK-006.
	unfollowable
)

func (s symlinkState) String() string {
	switch s {
	case followed:
		return "followed"
	case dangling:
		return "dangling"
	case unfollowable:
		return "unfollowable"
	}
	return "not a symlink"
}

// followSymlink says what path is and, when it is a symlink, what came of
// following it. The destination comes back whenever it can be read, so a message
// can name it, and the error comes back so a message can quote it.
func followSymlink(path string, stat statFunc) (symlinkState, string, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return notASymlink, "", nil
	}
	switch _, statErr := stat(path); {
	case statErr == nil:
		return followed, "", nil
	case os.IsNotExist(statErr):
		return dangling, readlink(path), nil
	default:
		return unfollowable, readlink(path), statErr
	}
}

// reason is the part of a failed Stat a reader needs: what went wrong, without
// the path it went wrong on.
//
// ⚠️ os.Stat wraps its error in an *fs.PathError carrying the path it was given,
// which is the link — while the message around this quotes the destination. Two
// different paths in one sentence, neither of them wrong, is a sentence nobody
// can read.
func reason(err error) string {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return pathError.Err.Error()
	}
	return err.Error()
}

// readlink names a symlink's destination, or says it could not be read.
func readlink(path string) string {
	destination, err := os.Readlink(path)
	if err != nil {
		return "an unreadable destination"
	}
	return destination
}
