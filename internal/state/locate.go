// Package state answers two questions that are about the repository rather
// than about any item: where it is, and whether it is behind.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/settings"
)

// EnvHome is the environment variable that names the repository.
const EnvHome = "PETKIT_HOME"

// ConfigRelPath is where `petkit init` records the repository, under the home
// directory.
const ConfigRelPath = ".config/petkit/config.json"

// Location is a resolved repository and the reason petkit believes in it.
type Location struct {
	Root string
	From string
}

// Config is the file `petkit init` writes.
type Config struct {
	Repo string `json:"repo"`
}

// ConfigPath is the config file's absolute path for a given home directory.
func ConfigPath(home string) string {
	return filepath.Join(home, filepath.FromSlash(ConfigRelPath))
}

// Find locates the repository: upwards from the working directory, then
// $PETKIT_HOME, then the path recorded by `petkit init`.
//
// The environment variable and the recorded path are explicit statements by the
// user, so when one of them is set and wrong petkit says so rather than quietly
// falling through to the next source.
func Find(workingDir, home string, env func(string) string) (Location, error) {
	if root, found := walkUp(workingDir); found {
		return Location{Root: root, From: "petkit.yaml found above the working directory"}, nil
	}

	if raw := env(EnvHome); raw != "" {
		root := manifest.ExpandTilde(raw, home)
		if !hasManifest(root) {
			return Location{}, fmt.Errorf(
				"%s is set to %s, but there is no %s there; point it at the petkit repository or unset it",
				EnvHome, root, manifest.FileName)
		}
		return Location{Root: root, From: EnvHome}, nil
	}

	configPath := ConfigPath(home)
	config, err := ReadConfig(configPath)
	switch {
	case err != nil:
		return Location{}, err
	case config != nil && config.Repo != "":
		root := manifest.ExpandTilde(config.Repo, home)
		if !hasManifest(root) {
			return Location{}, fmt.Errorf(
				"%s records the repository as %s, but there is no %s there; run `petkit init <path>` again",
				configPath, root, manifest.FileName)
		}
		return Location{Root: root, From: configPath}, nil
	}

	return Location{}, fmt.Errorf(
		"cannot find the petkit repository: no %s above %s, %s is not set, and nothing is recorded in %s — "+
			"run `petkit init /path/to/petkit` once, or run petkit from inside the repository",
		manifest.FileName, workingDir, EnvHome, configPath)
}

func walkUp(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if hasManifest(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func hasManifest(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, manifest.FileName))
	return err == nil && !info.IsDir()
}

// ReadConfig reads the recorded repository. A missing file is not an error: it
// only means `petkit init` has never run here.
func ReadConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w — delete it and run `petkit init <path>`", path, err)
	}
	return &config, nil
}

// Init records where the repository lives, so the other commands work from any
// directory. It refuses a path that is not a petkit repository, because a
// recorded path that resolves to nothing is worse than no record at all.
func Init(candidate, home string) (Location, error) {
	root := manifest.ExpandTilde(candidate, home)
	root, err := filepath.Abs(root)
	if err != nil {
		return Location{}, fmt.Errorf("cannot resolve %s: %w", candidate, err)
	}
	if !hasManifest(root) {
		return Location{}, fmt.Errorf(
			"%s is not a petkit repository: it has no %s — give the path of the checkout that does",
			root, manifest.FileName)
	}

	path := ConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Location{}, fmt.Errorf("cannot create %s: %w", filepath.Dir(path), err)
	}

	content, err := json.MarshalIndent(Config{Repo: root}, "", "  ")
	if err != nil {
		return Location{}, fmt.Errorf("cannot encode the config: %w", err)
	}
	content = append(content, '\n')

	if existing, err := os.ReadFile(path); err == nil {
		backup := path + ".petkit-backup"
		if err := os.WriteFile(backup, existing, 0o644); err != nil {
			return Location{}, fmt.Errorf("cannot write the backup %s: %w — nothing was changed", backup, err)
		}
	}
	if err := settings.WriteAtomic(path, content, 0o644); err != nil {
		return Location{}, err
	}
	return Location{Root: root, From: path}, nil
}
