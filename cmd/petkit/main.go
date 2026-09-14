// Command petkit keeps one machine's Claude Code setup in a git repository and
// installs it by symlink: the repository is the only copy, so editing a skill
// in the repository is editing the installed one.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/vukyn/petkit/internal/link"
	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/plugins"
	"github.com/vukyn/petkit/internal/settings"
	"github.com/vukyn/petkit/internal/state"
	"github.com/vukyn/petkit/internal/version"
)

// environment is everything the commands are told rather than discover. The
// home directory in particular is a parameter, never a read of $HOME inside the
// logic, which is what lets the tests run against a temporary machine.
type environment struct {
	home       string
	workingDir string
	env        func(string) string
	stdout     io.Writer
	stderr     io.Writer
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "petkit: cannot determine the home directory: %v\n", err)
		os.Exit(1)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "petkit: cannot determine the working directory: %v\n", err)
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:], environment{
		home:       home,
		workingDir: workingDir,
		env:        os.Getenv,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}))
}

func run(args []string, environment environment) int {
	if len(args) == 0 {
		usage(environment.stderr)
		return 2
	}

	var err error
	switch args[0] {
	case "version":
		err = commandVersion(environment)
	case "status":
		err = commandStatus(environment)
	case "sync":
		err = commandSync(environment, args[1:])
	case "doctor":
		err = commandDoctor(environment)
	case "settings":
		err = commandSettings(environment, args[1:])
	case "plugins":
		err = commandPlugins(environment, args[1:])
	case "check":
		err = commandCheck(environment)
	case "init":
		err = commandInit(environment, args[1:])
	case "help", "-h", "--help":
		usage(environment.stdout)
		return 0
	default:
		fmt.Fprintf(environment.stderr, "petkit: unknown command %q\n\n", args[0])
		usage(environment.stderr)
		return 2
	}

	if err != nil {
		if errors.Is(err, errUsage) {
			return 2
		}
		fmt.Fprintf(environment.stderr, "petkit: %v\n", err)
		return 1
	}
	return 0
}

var errUsage = errors.New("usage")

func usage(out io.Writer) {
	fmt.Fprint(out, `petkit — one machine's Claude Code setup, kept in a git repository and
installed by symlink.

  petkit version            the build version and the manifest's item count
  petkit status             one line per item: linked, missing, stale, conflict
  petkit sync [--dry-run]   create missing links, repoint stale ones
  petkit doctor             the checks that are not about one item
  petkit settings diff      what merging settings/fragment.json would change
  petkit settings apply     merge it, after a timestamped backup
  petkit plugins plan       the claude commands that would match the manifest
  petkit check              the tag this checkout is on, and the newest there is
  petkit init [path]        record where the repository lives

The repository is found by walking up from the working directory looking for
petkit.yaml, then $PETKIT_HOME, then the path recorded by petkit init.
`)
}

func locate(environment environment) (state.Location, error) {
	return state.Find(environment.workingDir, environment.home, environment.env)
}

func loadManifest(environment environment) (*manifest.Manifest, error) {
	location, err := locate(environment)
	if err != nil {
		return nil, err
	}
	return manifest.Load(location.Root)
}

func commandVersion(environment environment) error {
	fmt.Fprintf(environment.stdout, "petkit %s\n", version.Current())

	// The version of the binary is knowable even when the repository is not,
	// so a repository that cannot be found is a note here and not a failure.
	loaded, err := loadManifest(environment)
	if err != nil {
		fmt.Fprintf(environment.stdout, "manifest: not read (%v)\n", err)
		return nil
	}
	fmt.Fprintf(environment.stdout, "%d items in %s\n",
		len(loaded.Items), filepath.Join(loaded.Root, manifest.FileName))
	return nil
}

func commandStatus(environment environment) error {
	loaded, err := loadManifest(environment)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(environment.stdout, 0, 0, 2, ' ', 0)
	for _, itemState := range link.Inspect(loaded, environment.home) {
		target := manifest.CollapseHome(itemState.Target, environment.home)
		detail := ""
		switch itemState.Status {
		case link.Stale:
			detail = fmt.Sprintf("-> %s (want %s)",
				manifest.CollapseHome(itemState.Actual, environment.home),
				manifest.CollapseHome(itemState.Source, environment.home))
		case link.Conflict:
			detail = "(" + itemState.Detail + ")"
		}
		if detail != "" {
			detail = "\t" + detail
		}
		fmt.Fprintf(writer, "%s\t%s\t%s%s\n", itemState.Status, itemState.Item.ID, target, detail)
	}
	// status reports; it does not judge. Its exit code is always 0.
	return writer.Flush()
}

func commandSync(environment environment, args []string) error {
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	flags.SetOutput(environment.stderr)
	dryRun := flags.Bool("dry-run", false, "print the plan and change nothing")
	if err := flags.Parse(args); err != nil {
		return errUsage
	}

	loaded, err := loadManifest(environment)
	if err != nil {
		return err
	}

	actions := link.Plan(link.Inspect(loaded, environment.home))

	prefix := ""
	if *dryRun {
		prefix = "would "
	}
	writer := tabwriter.NewWriter(environment.stdout, 0, 0, 2, ' ', 0)
	for _, action := range actions {
		target := manifest.CollapseHome(action.State.Target, environment.home)
		source := manifest.CollapseHome(action.State.Source, environment.home)
		switch action.Kind {
		case link.Create:
			fmt.Fprintf(writer, "%screate\t%s\t%s -> %s\n", prefix, action.State.Item.ID, target, source)
		case link.Repoint:
			fmt.Fprintf(writer, "%srepoint\t%s\t%s -> %s (was %s)\n", prefix, action.State.Item.ID, target, source,
				manifest.CollapseHome(action.State.Actual, environment.home))
		case link.None:
			fmt.Fprintf(writer, "ok\t%s\t%s\n", action.State.Item.ID, target)
		case link.Refuse:
			fmt.Fprintf(writer, "conflict\t%s\t%s (%s)\n", action.State.Item.ID, target, action.State.Detail)
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	result, err := link.Apply(actions, *dryRun)
	if err != nil {
		return err
	}

	if result.Changed == 0 && len(result.Refused) == 0 {
		fmt.Fprintln(environment.stdout, "nothing to do; every item is already linked")
	} else if !*dryRun {
		fmt.Fprintf(environment.stdout, "%d change(s)\n", result.Changed)
	}

	if len(result.Refused) > 0 {
		fmt.Fprintln(environment.stderr, "\nrefused, because petkit removes symlinks and nothing else:")
		for _, action := range result.Refused {
			fmt.Fprintf(environment.stderr, "  %s is %s — for item %q\n",
				action.State.Target, action.State.Detail, action.State.Item.ID)
		}
		fmt.Fprintln(environment.stderr,
			"move each one aside yourself, then run petkit sync again")
		return fmt.Errorf("%d target(s) in the way; nothing was removed", len(result.Refused))
	}
	return nil
}

func commandDoctor(environment environment) error {
	loaded, err := loadManifest(environment)
	if err != nil {
		return err
	}

	findings := link.Doctor(loaded, environment.home)
	problems := 0
	for _, finding := range findings {
		if finding.Level == link.Problem {
			problems++
		}
		fmt.Fprintf(environment.stdout, "%-8s %s\n", finding.Level, finding.Message)
	}
	if len(findings) == 0 {
		fmt.Fprintln(environment.stdout, "nothing to report")
		return nil
	}
	fmt.Fprintf(environment.stdout,
		"\n%d problem(s), %d note(s). Notes are things petkit does not own.\n",
		problems, len(findings)-problems)
	if problems > 0 {
		return fmt.Errorf("%d problem(s) need a decision from you", problems)
	}
	return nil
}

func commandSettings(environment environment, args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(environment.stderr, "petkit settings: say diff or apply")
		return errUsage
	}

	location, err := locate(environment)
	if err != nil {
		return err
	}
	fragmentPath := filepath.Join(location.Root, "settings", "fragment.json")
	fragment, err := os.ReadFile(fragmentPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", fragmentPath, err)
	}
	livePath := filepath.Join(environment.home, ".claude", "settings.json")

	switch args[0] {
	case "diff":
		return settingsDiff(environment, livePath, fragment)
	case "apply":
		return settingsApply(environment, livePath, fragment)
	default:
		fmt.Fprintf(environment.stderr, "petkit settings: unknown subcommand %q; say diff or apply\n", args[0])
		return errUsage
	}
}

func settingsDiff(environment environment, livePath string, fragment []byte) error {
	live, err := os.ReadFile(livePath)
	if os.IsNotExist(err) {
		live = []byte("{}")
	} else if err != nil {
		return fmt.Errorf("cannot read %s: %w", livePath, err)
	}

	changes, err := settings.Diff(live, fragment)
	if err != nil {
		return err
	}
	shown := manifest.CollapseHome(livePath, environment.home)
	if len(changes) == 0 {
		fmt.Fprintf(environment.stdout, "%s already says what the fragment says\n", shown)
		return nil
	}
	fmt.Fprintf(environment.stdout, "%s would change in %d place(s):\n", shown, len(changes))
	for _, change := range settings.SortedPaths(changes) {
		if change.Added() {
			fmt.Fprintf(environment.stdout, "  + %s = %s\n", change.Path, change.New)
			continue
		}
		fmt.Fprintf(environment.stdout, "  ~ %s: %s -> %s\n", change.Path, change.Old, change.New)
	}
	fmt.Fprintln(environment.stdout,
		"\nevery key the fragment does not name keeps its value; nothing was written")
	return nil
}

func settingsApply(environment environment, livePath string, fragment []byte) error {
	result, err := settings.Apply(livePath, fragment, time.Now())
	if err != nil {
		return err
	}
	shown := manifest.CollapseHome(livePath, environment.home)
	if !result.Changed {
		fmt.Fprintf(environment.stdout, "%s already says what the fragment says; nothing was written\n", shown)
		return nil
	}
	if result.BackupPath != "" {
		fmt.Fprintf(environment.stdout, "backup %s\n", manifest.CollapseHome(result.BackupPath, environment.home))
	}
	fmt.Fprintf(environment.stdout, "wrote  %s\n", shown)
	return nil
}

func commandPlugins(environment environment, args []string) error {
	if len(args) == 0 || args[0] != "plan" {
		fmt.Fprintln(environment.stderr, "petkit plugins: say plan (petkit never installs anything itself)")
		return errUsage
	}

	location, err := locate(environment)
	if err != nil {
		return err
	}
	desired, err := plugins.LoadDesired(filepath.Join(location.Root, "settings", "plugins.json"))
	if err != nil {
		return err
	}
	live, err := plugins.LoadLive(filepath.Join(environment.home, ".claude", "plugins"))
	if err != nil {
		return err
	}

	plan := plugins.Build(desired, live)
	if len(plan.Commands) == 0 {
		fmt.Fprintln(environment.stdout, "this machine already has every marketplace and plugin in the manifest")
	} else {
		for _, command := range plan.Commands {
			fmt.Fprintf(environment.stdout, "%s\n    # %s\n", command.Line, command.Why)
		}
	}
	for _, note := range plan.Notes {
		fmt.Fprintf(environment.stdout, "# %s\n", note)
	}
	fmt.Fprintln(environment.stdout,
		"\nThese are commands to run by hand. petkit printed them and ran none of them.")
	return nil
}

func commandCheck(environment environment) error {
	location, err := locate(environment)
	if err != nil {
		return err
	}

	result, err := state.Check(location.Root, state.GitRunner)
	if err != nil {
		return err
	}

	fmt.Fprintf(environment.stdout, "repository %s\n", result.Root)
	if result.FetchNote != "" {
		fmt.Fprintf(environment.stdout, "fetch      %s\n", result.FetchNote)
	}
	switch {
	case result.Revision == "":
		fmt.Fprintln(environment.stdout, "checkout   no commits yet, so there is no version to be behind")
	case result.Tag == "" && !result.HasTags:
		fmt.Fprintf(environment.stdout, "checkout   %s, with no tags in the repository yet\n", result.Revision)
	case result.Tag == "":
		fmt.Fprintf(environment.stdout, "checkout   %s, with no tag in its history\n", result.Revision)
	case result.Commits == 0:
		fmt.Fprintf(environment.stdout, "checkout   on %s\n", result.Tag)
	default:
		fmt.Fprintf(environment.stdout, "checkout   %d commit(s) past %s (at %s)\n",
			result.Commits, result.Tag, result.Revision)
	}
	if result.HasTags {
		fmt.Fprintf(environment.stdout, "newest tag %s\n", result.NewestTag)
	}
	if result.Dirty {
		fmt.Fprintf(environment.stdout, "tree       dirty: %d path(s) changed\n", result.DirtyCount)
	} else {
		fmt.Fprintln(environment.stdout, "tree       clean")
	}
	if result.UpToDate() && !result.Dirty {
		fmt.Fprintln(environment.stdout, "\nthis checkout is the newest version there is")
	}
	return nil
}

func commandInit(environment environment, args []string) error {
	candidate := environment.workingDir
	if len(args) > 0 && args[0] != "" {
		candidate = args[0]
	}
	location, err := state.Init(candidate, environment.home)
	if err != nil {
		return err
	}
	fmt.Fprintf(environment.stdout, "recorded %s in %s\n", location.Root, location.From)
	fmt.Fprintln(environment.stdout, "the other commands now work from any directory")
	return nil
}
