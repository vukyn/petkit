package state

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
)

// DefaultSetupPath is where setup clones when it is given no path. It is a `~`
// path for the same reason a manifest target is one: the record that ends up in
// the config file must not be written by hand on every machine.
const DefaultSetupPath = "~/.petkit"

// SetupRequest is everything setup is told rather than discovers. The module
// path and the git runner are parameters so the derivation can be measured
// without a network, and the home directory is one so a test never touches the
// machine it runs on.
type SetupRequest struct {
	// Path is the directory to clone into. Empty means DefaultSetupPath, and
	// the difference matters: a machine that already has a repository is
	// refused unless a path was actually given.
	Path string

	Home string

	// ModulePath is debug.BuildInfo.Main.Path — the module this binary was
	// built from, which is the repository it came from.
	ModulePath string

	Run Runner
}

// SetupResult is what setup did: where it cloned from, and the record it left
// behind.
type SetupResult struct {
	CloneURL string
	Target   string
	Location Location
}

// CloneURL turns the module path of the running binary into the repository to
// clone.
//
// ⚠️ There is no URL constant here on purpose. `go install <module>/cmd/petkit`
// stamps the module it installed into the build info, so a fork clones the fork
// — which is the one thing a hard-coded address cannot do, and it would fail
// silently by cloning something that works.
func CloneURL(modulePath string) (string, error) {
	if modulePath == "" {
		return "", fmt.Errorf(
			"this build carries no module path, so setup cannot tell which repository it came from " +
				"(a binary built outside module mode) — clone it yourself with `git clone <the petkit repository> ~/.petkit`, " +
				"then run `petkit init ~/.petkit`")
	}
	return "https://" + modulePath, nil
}

// Setup clones the repository this binary was built from and records where it
// went, so that a machine with nothing but the binary has everything the other
// commands need.
//
// It creates no symlinks: `sync` owns that, and setup's caller decides whether
// to run it.
//
// ⚠️ Setup removes nothing, ever. The target must be missing or an empty
// directory; anything else is refused by name and left exactly as it was. This
// is `sync`'s conflict rule applied to the one directory setup writes.
func Setup(request SetupRequest) (SetupResult, error) {
	url, err := CloneURL(request.ModulePath)
	if err != nil {
		return SetupResult{}, err
	}

	givenPath := request.Path != ""
	candidate := request.Path
	if !givenPath {
		candidate = DefaultSetupPath
	}
	target, err := filepath.Abs(manifest.ExpandTilde(candidate, request.Home, ospath.Current()))
	if err != nil {
		return SetupResult{}, fmt.Errorf("cannot resolve %s: %w", candidate, err)
	}

	if err := refuseIfAlreadySetUp(request.Home, givenPath, target); err != nil {
		return SetupResult{}, err
	}
	if err := refuseIfOccupied(target); err != nil {
		return SetupResult{}, err
	}

	// git creates the target itself; the parent has to exist for the command to
	// have somewhere to run.
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return SetupResult{}, fmt.Errorf("cannot create %s: %w", parent, err)
	}

	// No --depth: `petkit check` compares tags, and a shallow clone has none.
	if _, err := request.Run(parent, "clone", url, target); err != nil {
		return SetupResult{}, fmt.Errorf("cannot clone %s into %s: %w", url, target, err)
	}

	// The record is `petkit init`'s, not a second one: a machine set up by
	// setup and a machine set up by hand are the same machine afterwards.
	location, err := Init(target, request.Home)
	if err != nil {
		return SetupResult{}, err
	}

	return SetupResult{CloneURL: url, Target: target, Location: location}, nil
}

// refuseIfAlreadySetUp stops a second clone of the thing whose whole point is
// that there is one. A record that no longer resolves is not a set-up machine,
// so it does not stop anything — `Find` already says that record is broken, and
// setup is a reasonable way to fix it.
func refuseIfAlreadySetUp(home string, givenPath bool, target string) error {
	configPath := ConfigPath(home)
	config, err := ReadConfig(configPath)
	if err != nil {
		return err
	}
	if config == nil || config.Repo == "" {
		return nil
	}
	recorded := manifest.ExpandTilde(config.Repo, home, ospath.Current())
	if !hasManifest(recorded) {
		return nil
	}
	// A path argument naming somewhere else is an explicit request for a second
	// clone, and explicit is the difference between a decision and an accident.
	if givenPath && filepath.Clean(recorded) != filepath.Clean(target) {
		return nil
	}
	return fmt.Errorf(
		"this machine already has a petkit repository at %s, recorded in %s — "+
			"run `petkit sync` to install from it, or give setup a path if you really want a second clone elsewhere",
		recorded, configPath)
}

// refuseIfOccupied is the whole of setup's destructiveness rule: the target is
// missing, or it is an empty directory, or setup does not touch it.
func refuseIfOccupied(target string) error {
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot look at %s: %w", target, err)
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"%s already exists and is not a directory; petkit setup removes nothing — "+
				"move it aside yourself, or give setup another path",
			target)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", target, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf(
			"%s already exists and has %d entr%s in it; petkit setup removes nothing — "+
				"empty it yourself, or give setup another path",
			target, len(entries), plural(len(entries)))
	}
	return nil
}

func plural(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}
