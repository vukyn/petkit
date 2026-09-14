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
	case filepath.IsAbs(source):
		return []string{fmt.Sprintf(
			"item %q: source %q is absolute; sources are relative to the repository root", label, source)}
	}
	clean := filepath.Clean(source)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return []string{fmt.Sprintf(
			"item %q: source %q escapes the repository; sources must stay inside %s", label, source, root)}
	}
	return nil
}

// SourcePath is the item's absolute path inside the repository.
func (i Item) SourcePath(root string) string {
	return filepath.Join(root, filepath.Clean(i.Source))
}

// TargetPath is the item's absolute path on the machine, with ~ expanded
// against the given home directory.
func (i Item) TargetPath(home string) string {
	return ExpandTilde(i.Target, home)
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

// CollapseHome is ExpandTilde's inverse, used only for printing: a path shown
// as ~/.claude/skills/x is the path the manifest names.
func CollapseHome(path, home string) string {
	home = filepath.Clean(home)
	clean := filepath.Clean(path)
	switch {
	case clean == home:
		return "~"
	case strings.HasPrefix(clean, home+string(filepath.Separator)):
		return "~" + clean[len(home):]
	default:
		return clean
	}
}
