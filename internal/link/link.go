// Package link compares the manifest against the filesystem and makes the
// filesystem match it — by symlink, and only by symlink.
//
// The one destructive operation here is removing a symlink in order to repoint
// it. Anything at a target that is not a symlink is a conflict: petkit names it
// and stops. A regular file or a real directory is always somebody else's.
package link

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vukyn/petkit/internal/manifest"
)

// Status is what petkit found at an item's target.
type Status string

const (
	// Linked: the target is a symlink resolving to the item's source.
	Linked Status = "linked"
	// Missing: there is nothing at the target.
	Missing Status = "missing"
	// Stale: the target is a symlink pointing somewhere else.
	Stale Status = "stale"
	// Conflict: the target exists and is not a symlink.
	Conflict Status = "conflict"
)

// ItemState is one manifest item measured against the machine.
type ItemState struct {
	Item   manifest.Item
	Status Status
	Source string // absolute, inside the repository
	Target string // absolute, ~ already expanded
	Actual string // where an existing symlink points (Stale only)
	Detail string // what is in the way (Conflict only)
}

// Inspect measures every item. It reads the filesystem and changes nothing.
func Inspect(m *manifest.Manifest, home string) []ItemState {
	states := make([]ItemState, 0, len(m.Items))
	for _, item := range m.Items {
		states = append(states, inspectOne(item, m.Root, home))
	}
	return states
}

func inspectOne(item manifest.Item, root, home string) ItemState {
	state := ItemState{
		Item:   item,
		Source: item.SourcePath(root),
		Target: item.TargetPath(home),
	}

	info, err := os.Lstat(state.Target)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		state.Status = Missing
		return state
	case err != nil:
		// Unreadable is not the same as absent: petkit cannot prove what is
		// there, so it treats it as something it must not touch.
		state.Status = Conflict
		state.Detail = err.Error()
		return state
	case info.Mode()&fs.ModeSymlink == 0:
		state.Status = Conflict
		state.Detail = describe(info)
		return state
	}

	destination, err := os.Readlink(state.Target)
	if err != nil {
		state.Status = Conflict
		state.Detail = fmt.Sprintf("symlink that cannot be read: %v", err)
		return state
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(filepath.Dir(state.Target), destination)
	}
	state.Actual = filepath.Clean(destination)

	if samePath(state.Actual, state.Source) {
		state.Status = Linked
	} else {
		state.Status = Stale
	}
	return state
}

func describe(info fs.FileInfo) string {
	if info.IsDir() {
		return "a real directory, not a symlink"
	}
	return "a regular file, not a symlink"
}

// samePath compares two paths first literally and then, only if that fails,
// through symlink resolution — so /var/folders/... and /private/var/folders/...
// on macOS are recognised as one place.
func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	resolvedA, errA := filepath.EvalSymlinks(a)
	resolvedB, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil {
		return false
	}
	return resolvedA == resolvedB
}

// ActionKind is what sync intends to do about one item.
type ActionKind string

const (
	// None: the target already resolves to the source.
	None ActionKind = "none"
	// Create: nothing is there; make the link (and any parent directory).
	Create ActionKind = "create"
	// Repoint: a symlink points elsewhere; replace it.
	Repoint ActionKind = "repoint"
	// Refuse: something that is not a symlink is in the way.
	Refuse ActionKind = "refuse"
)

// Action pairs a measured item with the intent.
type Action struct {
	State ItemState
	Kind  ActionKind
}

// Plan turns measurements into intents. It decides nothing destructive: the
// only removal it can ever propose is of a symlink.
func Plan(states []ItemState) []Action {
	actions := make([]Action, 0, len(states))
	for _, state := range states {
		kind := None
		switch state.Status {
		case Missing:
			kind = Create
		case Stale:
			kind = Repoint
		case Conflict:
			kind = Refuse
		}
		actions = append(actions, Action{State: state, Kind: kind})
	}
	return actions
}

// Result is what Apply did, or would have done.
type Result struct {
	Changed   int
	Refused   []Action
	DryRun    bool
	Performed []Action
}

// Apply carries out a plan. With dryRun set it touches nothing and reports the
// same numbers, so `--dry-run` and the real run cannot disagree.
func Apply(actions []Action, dryRun bool) (Result, error) {
	result := Result{DryRun: dryRun}
	for _, action := range actions {
		switch action.Kind {
		case Create:
			if !dryRun {
				if err := createLink(action.State); err != nil {
					return result, err
				}
			}
			result.Changed++
			result.Performed = append(result.Performed, action)
		case Repoint:
			if !dryRun {
				if err := repointLink(action.State); err != nil {
					return result, err
				}
			}
			result.Changed++
			result.Performed = append(result.Performed, action)
		case Refuse:
			result.Refused = append(result.Refused, action)
		}
	}
	return result, nil
}

func createLink(state ItemState) error {
	parent := filepath.Dir(state.Target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("cannot create the directory %s for item %q: %w", parent, state.Item.ID, err)
	}
	if err := os.Symlink(state.Source, state.Target); err != nil {
		return fmt.Errorf("cannot link %s -> %s for item %q: %w",
			state.Target, state.Source, state.Item.ID, err)
	}
	return nil
}

func repointLink(state ItemState) error {
	// Proof of ownership, taken again immediately before the removal rather
	// than trusted from the earlier measurement: petkit removes symlinks and
	// nothing else, so it re-checks that a symlink is still what is there.
	info, err := os.Lstat(state.Target)
	if err != nil {
		return fmt.Errorf("cannot inspect %s for item %q: %w", state.Target, state.Item.ID, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return fmt.Errorf(
			"%s is no longer a symlink; refusing to remove it — inspect it yourself and move it aside "+
				"if item %q should own that path", state.Target, state.Item.ID)
	}
	if err := os.Remove(state.Target); err != nil {
		return fmt.Errorf("cannot remove the symlink %s for item %q: %w", state.Target, state.Item.ID, err)
	}
	if err := os.Symlink(state.Source, state.Target); err != nil {
		return fmt.Errorf("cannot link %s -> %s for item %q: %w",
			state.Target, state.Source, state.Item.ID, err)
	}
	return nil
}
