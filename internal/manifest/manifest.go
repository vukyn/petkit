// Package manifest reads and validates petkit.yaml, the single description of
// what a machine should have and where each piece goes.
//
// Nothing in this package touches the home directory: the home directory is a
// parameter everywhere it is needed, so the same code runs against a real
// machine and against a temporary one in a test.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vukyn/petkit/internal/ospath"
)

// FileName is the manifest's name; finding it is what marks a repository root.
const FileName = "petkit.yaml"

// SupportedVersion is the only manifest schema this binary understands.
const SupportedVersion = 1

// Item is one thing the repository owns and one place it belongs.
type Item struct {
	ID     string `yaml:"id"`
	Kind   string `yaml:"kind"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Note   string `yaml:"note"`
}

// Manifest is the parsed petkit.yaml together with the root it was read from.
type Manifest struct {
	Version int    `yaml:"version"`
	Items   []Item `yaml:"items"`

	// Root is the repository directory holding this manifest. It is filled in
	// by Load/Parse and never read from the file itself.
	Root string `yaml:"-"`
}

// ValidationError collects every problem found in one manifest, so a bad file
// is reported once and in full rather than one problem per run.
type ValidationError struct {
	Path     string
	Problems []string
}

func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is not usable:", e.Path)
	for _, problem := range e.Problems {
		fmt.Fprintf(&b, "\n  - %s", problem)
	}
	b.WriteString("\nfix the manifest and run the command again")
	return b.String()
}

// Load reads and validates the manifest at the root of a repository.
func Load(root string) (*Manifest, error) {
	path := filepath.Join(root, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	return Parse(data, root)
}

// Parse validates manifest bytes as if they had been read from root.
func Parse(data []byte, root string) (*Manifest, error) {
	var parsed Manifest
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", filepath.Join(root, FileName), err)
	}
	parsed.Root = root
	if err := parsed.Validate(); err != nil {
		return nil, err
	}
	return &parsed, nil
}

// Validate refuses every manifest that could make a command do something
// surprising. It runs before any action, so a refusal costs nothing.
func (m *Manifest) Validate() error {
	var problems []string

	if m.Version != SupportedVersion {
		problems = append(problems, fmt.Sprintf(
			"version is %d; this petkit understands version %d only", m.Version, SupportedVersion))
	}
	if len(m.Items) == 0 {
		problems = append(problems, "items is empty; a manifest with no items installs nothing")
	}

	idOwner := make(map[string]int, len(m.Items))
	targetOwner := make(map[string]string, len(m.Items))

	for index, item := range m.Items {
		label := item.ID
		if label == "" {
			label = fmt.Sprintf("items[%d]", index)
			problems = append(problems, fmt.Sprintf(
				"%s: id is empty; every item needs a stable id so errors can name it", label))
		} else if first, duplicate := idOwner[item.ID]; duplicate {
			problems = append(problems, fmt.Sprintf(
				"duplicate id %q (items[%d] and items[%d]); ids must be unique", item.ID, first, index))
		} else {
			idOwner[item.ID] = index
		}

		problems = append(problems, validateTarget(label, item.Target, targetOwner)...)
		problems = append(problems, validateSource(label, item.Source, m.Root)...)
	}

	if len(problems) == 0 {
		return nil
	}
	return &ValidationError{Path: filepath.Join(m.Root, FileName), Problems: problems}
}

func validateTarget(label, target string, owner map[string]string) []string {
	switch {
	case target == "":
		return []string{fmt.Sprintf("item %q: target is empty; write it as ~/...", label)}
	case !strings.HasPrefix(target, "~/"):
		return []string{fmt.Sprintf(
			"item %q: target %q must start with \"~/\"; an absolute or relative target "+
				"would make petkit.yaml machine-specific, which it may not be", label, target)}
	case strings.Contains(target, `\`):
		return []string{backslashProblem(label, "target", target)}
	}
	// ⚠️ `~/` is a prefix, not a fence: `~/../elsewhere` starts with it and lands
	// outside the home directory once expanded. Refuse it here rather than leaving
	// it to doctor — sync would have created the link first.
	if rest := filepath.Clean(target[len("~/"):]); rest == ".." || strings.HasPrefix(rest, ".."+string(filepath.Separator)) {
		return []string{fmt.Sprintf(
			"item %q: target %q climbs out of the home directory; a target must stay under ~/", label, target)}
	}
	key := filepath.Clean(target)
	if other, duplicate := owner[key]; duplicate {
		return []string{fmt.Sprintf(
			"items %q and %q both claim target %q; a target belongs to one item only", other, label, target)}
	}
	owner[key] = label
	return nil
}

func validateSource(label, source, root string) []string {
	switch {
	case source == "":
		return []string{fmt.Sprintf("item %q: source is empty; name a path inside the repository", label)}
	// ⚠️ ospath.IsAbsAnywhere, not filepath.IsAbs. filepath is compiled for the
	// host and the two platforms disagree in both directions, so the host's
	// answer made the same manifest legal on one machine and illegal on the
	// other — MFST-006. The rule has to be the manifest's, not the machine's.
	case ospath.IsAbsAnywhere(source):
		return []string{fmt.Sprintf(
			"item %q: source %q is absolute; sources are relative to the repository root", label, source)}
	case strings.Contains(source, `\`):
		return []string{backslashProblem(label, "source", source)}
	}
	clean := filepath.Clean(source)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return []string{fmt.Sprintf(
			"item %q: source %q escapes the repository; sources must stay inside %s", label, source, root)}
	}
	return nil
}

// backslashProblem refuses a path written with Windows separators.
//
// ⚠️ This is refused on every platform, not only on Unix, and the reason is that
// the manifest travels. `~/..\elsewhere` is a path that climbs out of the home
// directory on Windows and an ordinary file called `..\elsewhere` on macOS, so
// the escape check would pass on the machine that wrote the manifest and the
// escape would happen on the machine that read it. One spelling, checked once,
// meaning the same thing everywhere.
func backslashProblem(label, field, value string) string {
	return fmt.Sprintf(
		"item %q: %s %q contains a backslash; write every path in %s with \"/\" — a backslash "+
			"separates components on Windows and is an ordinary character in a file name everywhere "+
			"else, so the same manifest would mean two different things",
		label, field, value, FileName)
}

// SourcePath is the item's absolute path inside the repository. goos is the
// platform the path is being built for; the manifest writes a source with "/"
// on every machine, so that is where the separator is decided.
func (i Item) SourcePath(root, goos string) string {
	return filepath.Join(root, ospath.FromSlash(i.Source, goos))
}

// TargetPath is the item's absolute path on the machine.
func (i Item) TargetPath(layout Layout) string {
	return layout.Resolve(i.Target)
}

// ExpandTilde expands a leading ~ against home. It is the only templating
// petkit does: no $HOME, no other variables.
func ExpandTilde(path, home string) string {
	switch {
	case path == "~":
		return filepath.Clean(home)
	case strings.HasPrefix(path, "~/"):
		return filepath.Join(home, path[len("~/"):])
	default:
		return path
	}
}

// EnvConfigDir is the variable Claude Code itself honours for the directory it
// keeps its configuration in.
//
// ⚠️ petkit used to hard-code ~/.claude. On a machine that sets this, that is a
// directory nothing reads: the links would be installed correctly and Claude
// Code would never look at them. It is not a Windows defect — it is wrong the
// same way on every platform.
const EnvConfigDir = "CLAUDE_CONFIG_DIR"

// ConfigDirName is where Claude Code keeps its configuration when
// CLAUDE_CONFIG_DIR says nothing.
const ConfigDirName = ".claude"

// configDirTarget is the manifest spelling of that directory. A target under it
// follows the variable; every other target is relative to the home directory.
const configDirTarget = "~/" + ConfigDirName

// Layout is the machine a manifest is being applied to: what `~` means, what
// `~/.claude` means, and which operating system's path rules to use.
//
// ⚠️ GOOS is a field rather than a read of runtime.GOOS for the reason the whole
// of internal/ospath exists: it lets a test on macOS resolve and print paths the
// way Windows would, in the same process, with no build tag.
type Layout struct {
	Home   string
	Config string
	GOOS   string
}

// NewLayout describes a machine: its home directory, the platform it runs, and
// the environment it was started with. CLAUDE_CONFIG_DIR wins when it is set and
// not empty, and a `~` inside it is expanded like any other `~`.
func NewLayout(home, goos string, env func(string) string) Layout {
	layout := Layout{
		Home:   home,
		Config: filepath.Join(home, ConfigDirName),
		GOOS:   goos,
	}
	if env == nil {
		return layout
	}
	if raw := env(EnvConfigDir); raw != "" {
		layout.Config = filepath.Clean(ExpandTilde(raw, home))
	}
	return layout
}

// Resolve turns a manifest target into a path on this machine. `~` is the home
// directory and `~/.claude` is the configuration directory — the same place
// unless CLAUDE_CONFIG_DIR moved it.
//
// The remainder is converted from the manifest's "/" to the platform's
// separator here, which is the one boundary where a target stops being text the
// manifest wrote and becomes a path the filesystem will be asked about.
func (l Layout) Resolve(target string) string {
	switch {
	case target == "~":
		return filepath.Clean(l.Home)
	case target == configDirTarget:
		return filepath.Clean(l.Config)
	case strings.HasPrefix(target, configDirTarget+"/"):
		return l.join(l.Config, target[len(configDirTarget)+1:])
	case strings.HasPrefix(target, "~/"):
		return l.join(l.Home, target[len("~/"):])
	default:
		return target
	}
}

// Display is Resolve's inverse, used only for printing: a path shown as
// ~/.claude/skills/x is the path the manifest names, spelled the way the
// manifest spells it — with "/", on every platform.
//
// A path that is not under the home directory is printed as it is. That is the
// honest answer for a configuration directory moved outside `~`: writing it back
// as `~/.claude/...` would name a place the file is not.
func (l Layout) Display(path string) string {
	home := filepath.Clean(l.Home)
	clean := filepath.Clean(path)
	rest, under := ospath.Under(clean, home, l.GOOS)
	switch {
	case !under:
		return clean
	case rest == "":
		return "~"
	default:
		return "~/" + ospath.ToSlash(rest, l.GOOS)
	}
}

// SkillsDir is where doctor surveys for entries petkit does not manage.
func (l Layout) SkillsDir() string {
	return filepath.Join(l.Config, "skills")
}

// join sticks a "/"-written remainder onto an absolute base.
func (l Layout) join(base, rest string) string {
	return filepath.Join(base, ospath.FromSlash(rest, l.GOOS))
}
