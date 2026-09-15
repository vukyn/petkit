// Command petkit keeps one machine's Claude Code setup in a git repository and
// installs it by symlink: the repository is the only copy, so editing a skill
// in the repository is editing the installed one.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/vukyn/petkit/internal/link"
	"github.com/vukyn/petkit/internal/manifest"
	"github.com/vukyn/petkit/internal/ospath"
	"github.com/vukyn/petkit/internal/plugins"
	"github.com/vukyn/petkit/internal/settings"
	"github.com/vukyn/petkit/internal/state"
	"github.com/vukyn/petkit/internal/version"
)

// environment is everything the commands are told rather than discover. The
// home directory in particular is a parameter, never a read of $HOME inside the
// logic, which is what lets the tests run against a temporary machine.
type environment struct {
	// layout is the machine: what `~` means, what `~/.claude` means once
	// CLAUDE_CONFIG_DIR has had its say, and which platform's path rules
	// apply. Home is read from it rather than kept twice.
	layout     manifest.Layout
	workingDir string
	env        func(string) string
	stdout     io.Writer
	stderr     io.Writer

	// git runs git, and modulePath is the module this binary was built from.
	// Both are parameters for the same reason the home directory is one: setup
	// is measured without a network and without cloning anything.
	git        state.Runner
	modulePath string
}

// environmentKey is where the environment is parked on the root command. urfave
// hands some hooks nothing but a *cli.Command, so the command graph is the only
// place they can find it.
const environmentKey = "environment"

func init() {
	// urfave renders help and the version through package-level hooks. Both are
	// replaced so that the framework prints what petkit printed before it: the
	// help below, and `petkit version`.
	cli.HelpPrinter = printHelp
	cli.VersionPrinter = printVersion
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
		layout:     manifest.NewLayout(home, ospath.Current(), os.Getenv),
		workingDir: workingDir,
		env:        os.Getenv,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		git:        state.GitRunner,
		modulePath: version.ModulePath(),
	}))
}

// run turns a command line into an exit code. urfave parses and dispatches;
// what a failure is worth is still decided here, because the exit codes are
// part of the interface: 2 for a usage error, 1 for a command that ran and
// failed, 0 otherwise.
func run(args []string, environment environment) int {
	command := newRootCommand(environment)

	// urfave expects the program name in position zero, the way os.Args has it.
	err := command.Run(context.Background(), append([]string{"petkit"}, args...))
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

func newRootCommand(environment environment) *cli.Command {
	root := &cli.Command{
		Name:  "petkit",
		Usage: "one machine's Claude Code setup, kept in a git repository and installed by symlink",

		// The version is the git tag, read out of the build info at run time.
		// There is no constant here to keep in step.
		Version: version.Current(),

		Writer:    environment.stdout,
		ErrWriter: environment.stderr,
		Metadata:  map[string]any{environmentKey: environment},

		// An error is turned into an exit code by run. urfave must not also
		// call os.Exit for it, which would take the tests down with it.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		OnUsageError:   usageError(environment),

		// Reached when the first argument names no command, and when there is
		// no argument at all. Both are usage errors, and both print the help.
		Action: func(_ context.Context, command *cli.Command) error {
			if name := command.Args().First(); name != "" {
				fmt.Fprintf(environment.stderr, "petkit: unknown command %q\n\n", name)
			}
			printHelp(environment.stderr, "", command)
			return errUsage
		},

		// The order is the order the help lists them in.
		Commands: []*cli.Command{
			{
				Name:         "setup",
				Usage:        "clone the repository this binary came from, and record where it went",
				UsageText:    "petkit setup [path] [--sync]",
				OnUsageError: usageError(environment),
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "sync",
						Usage: "install the links too, instead of printing the command that would",
					},
				},
				Action: func(_ context.Context, command *cli.Command) error {
					return commandSetup(environment, command.Args().First(), command.Bool("sync"))
				},
			},
			{
				Name:         "version",
				Usage:        "the build version and the manifest's item count",
				OnUsageError: usageError(environment),
				Action: func(context.Context, *cli.Command) error {
					return commandVersion(environment)
				},
			},
			{
				Name:         "status",
				Usage:        "one line per item: linked, missing, stale, conflict",
				OnUsageError: usageError(environment),
				Action: func(context.Context, *cli.Command) error {
					return commandStatus(environment)
				},
			},
			{
				Name:         "sync",
				Usage:        "create missing links, repoint stale ones",
				UsageText:    "petkit sync [--dry-run]",
				OnUsageError: usageError(environment),
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "print the plan and change nothing",
					},
				},
				Action: func(_ context.Context, command *cli.Command) error {
					return commandSync(environment, command.Bool("dry-run"))
				},
			},
			{
				Name:         "doctor",
				Usage:        "the checks that are not about one item",
				OnUsageError: usageError(environment),
				Action: func(context.Context, *cli.Command) error {
					return commandDoctor(environment)
				},
			},
			{
				Name:         "settings",
				Usage:        "what merging settings/fragment.json would change, or the merge itself",
				UsageText:    "petkit settings <diff|apply>",
				OnUsageError: usageError(environment),
				Commands: []*cli.Command{
					{
						Name:         "diff",
						Usage:        "what merging settings/fragment.json would change",
						UsageText:    "petkit settings diff",
						OnUsageError: usageError(environment),
						Action: func(context.Context, *cli.Command) error {
							return withFragment(environment, settingsDiff)
						},
					},
					{
						Name:         "apply",
						Usage:        "merge it, after a timestamped backup",
						UsageText:    "petkit settings apply",
						OnUsageError: usageError(environment),
						Action: func(context.Context, *cli.Command) error {
							return withFragment(environment, settingsApply)
						},
					},
				},
				// Reached only when the subcommand is missing or unnamed: a
				// recognised one is dispatched before this runs.
				Action: func(_ context.Context, command *cli.Command) error {
					if name := command.Args().First(); name != "" {
						fmt.Fprintf(environment.stderr,
							"petkit settings: unknown subcommand %q; say diff or apply\n", name)
						return errUsage
					}
					fmt.Fprintln(environment.stderr, "petkit settings: say diff or apply")
					return errUsage
				},
			},
			{
				Name:         "plugins",
				Usage:        "the claude commands that would match the manifest",
				UsageText:    "petkit plugins <plan|capture>",
				OnUsageError: usageError(environment),
				Commands: []*cli.Command{
					{
						Name:         "plan",
						Usage:        "the claude commands that would match the manifest",
						UsageText:    "petkit plugins plan",
						OnUsageError: usageError(environment),
						Action: func(context.Context, *cli.Command) error {
							return commandPluginsPlan(environment)
						},
					},
					{
						Name:         "capture",
						Usage:        "rewrite settings/plugins.json from what this machine has",
						UsageText:    "petkit plugins capture",
						OnUsageError: usageError(environment),
						Action: func(context.Context, *cli.Command) error {
							return commandPluginsCapture(environment)
						},
					},
				},
				Action: func(_ context.Context, command *cli.Command) error {
					if name := command.Args().First(); name != "" {
						fmt.Fprintf(environment.stderr,
							"petkit plugins: unknown subcommand %q; say plan or capture\n", name)
						return errUsage
					}
					fmt.Fprintln(environment.stderr,
						"petkit plugins: say plan or capture (petkit never installs anything itself)")
					return errUsage
				},
			},
			{
				Name:         "check",
				Usage:        "the tag this checkout is on, and the newest there is",
				OnUsageError: usageError(environment),
				Action: func(context.Context, *cli.Command) error {
					return commandCheck(environment)
				},
			},
			{
				Name:         "init",
				Usage:        "record where the repository lives",
				UsageText:    "petkit init [path]",
				OnUsageError: usageError(environment),
				Action: func(_ context.Context, command *cli.Command) error {
					return commandInit(environment, command.Args().First())
				},
			},
		},
	}
	return root
}

// usageError reports a command line urfave could not parse the way an unknown
// command is reported: named, on stderr, and worth exit code 2 rather than the
// 1 a command that ran and failed is worth.
func usageError(environment environment) cli.OnUsageErrorFunc {
	return func(_ context.Context, command *cli.Command, err error, _ bool) error {
		fmt.Fprintf(environment.stderr, "%s: %v\n", command.FullName(), err)
		return errUsage
	}
}

// printVersion answers `petkit --version`. `petkit version` is the authority on
// what a version looks like, so the flag runs the same code rather than urfave's
// own one-liner.
func printVersion(command *cli.Command) {
	if recorded, ok := command.Root().Metadata[environmentKey].(environment); ok {
		_ = commandVersion(recorded)
	}
}

// printHelp replaces urfave's help for the root command only; a subcommand's
// help is still urfave's.
func printHelp(out io.Writer, template string, data any) {
	command, ok := data.(*cli.Command)
	if !ok || command.Root() != command {
		cli.DefaultPrintHelp(out, template, data)
		return
	}
	usage(out, command)
}

// usage prints the summary of the whole tool. The command list is read out of
// the command graph rather than written out here, so a command that is dropped
// from the graph disappears from the help with it — a help text that is a
// literal is a second list nobody keeps in step.
func usage(out io.Writer, root *cli.Command) {
	fmt.Fprint(out, `petkit — one machine's Claude Code setup, kept in a git repository and
installed by symlink.

`)
	writer := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	for _, line := range usageLines(root) {
		fmt.Fprintf(writer, "  %s\t%s\n", line.invocation, line.summary)
	}
	_ = writer.Flush()
	fmt.Fprint(out, `
The repository is found by walking up from the working directory looking for
petkit.yaml, then $PETKIT_HOME, then the path recorded by petkit init.
`)
}

// usageLine is one row of the summary: how the command is typed, and what it
// does.
type usageLine struct {
	invocation string
	summary    string
}

// usageLines flattens the command graph one level: a command that has
// subcommands is listed as its subcommands, because that is how it is typed.
func usageLines(root *cli.Command) []usageLine {
	var lines []usageLine
	for _, command := range root.VisibleCommands() {
		subcommands := command.VisibleCommands()
		if len(subcommands) == 0 {
			lines = append(lines, usageLine{
				invocation: invocation(command, "petkit "+command.Name),
				summary:    command.Usage,
			})
			continue
		}
		for _, subcommand := range subcommands {
			lines = append(lines, usageLine{
				invocation: invocation(subcommand, "petkit "+command.Name+" "+subcommand.Name),
				summary:    subcommand.Usage,
			})
		}
	}
	return lines
}

// invocation is the command's UsageText where it has one, because that is where
// the argument shape is written down.
func invocation(command *cli.Command, fallback string) string {
	if command.UsageText != "" {
		return command.UsageText
	}
	return fallback
}

func locate(environment environment) (state.Location, error) {
	return state.Find(environment.workingDir, environment.layout.Home, environment.env)
}

func loadManifest(environment environment) (*manifest.Manifest, error) {
	_, loaded, err := loadFrom(environment)
	return loaded, err
}

// loadFrom resolves the repository and reads its manifest, keeping the location
// as well: every command that acts on the repository has to say which one it
// resolved, and that answer is thrown away by the time the manifest is in hand.
func loadFrom(environment environment) (state.Location, *manifest.Manifest, error) {
	location, err := locate(environment)
	if err != nil {
		return state.Location{}, nil, err
	}
	loaded, err := manifest.Load(location.Root)
	if err != nil {
		return location, nil, err
	}
	return location, loaded, nil
}

// reportRepository names the repository a command is acting on and how it was
// chosen. One line, always, on every command where acting on the wrong
// repository costs something.
//
// ⚠️ The reason it is unconditional is the case in the second branch. The
// repository is resolved by walking up from the working directory FIRST, so a
// machine set up at one path, whose shell happens to sit inside another clone,
// gets the other one — and `sync` then repoints every link into it. Nothing was
// wrong from petkit's side and nothing said anything, so the only way to notice
// was to read the paths in the plan. When the two disagree the line says both:
// what was used, and what `petkit init` recorded.
func reportRepository(environment environment, location state.Location) {
	used := environment.layout.Display(location.Root)
	if location.Recorded != "" &&
		!ospath.Equal(location.Recorded, location.Root, environment.layout.GOOS) {
		fmt.Fprintf(environment.stdout, "%-10s %s (%s) — but petkit init recorded %s\n",
			"repository", used, location.How, environment.layout.Display(location.Recorded))
		return
	}
	fmt.Fprintf(environment.stdout, "%-10s %s (%s)\n", "repository", used, location.How)
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
	location, loaded, err := loadFrom(environment)
	if err != nil {
		return err
	}
	reportRepository(environment, location)
	if err := statusOf(environment, loaded); err != nil {
		return err
	}
	reportSettings(environment, location.Root)
	return nil
}

// reportSettings is the one line that says whether the settings file on this
// machine still says what the fragment says.
//
// ⚠️ It reports and never fails. `settings diff` answers this question when
// asked and nothing asks it, so a machine where the fragment was applied months
// ago and has been edited by hand since reports `linked` on every skill and is
// still not the machine the repository describes. Drift is a fact about a
// machine, not a broken command: status keeps its exit code of 0 on every path
// through here, including the paths where the comparison itself cannot be made.
func reportSettings(environment environment, root string) {
	livePath := filepath.Join(environment.layout.Config, "settings.json")
	shown := environment.layout.Display(livePath)

	fragmentPath := filepath.Join(root, "settings", "fragment.json")
	fragment, err := os.ReadFile(fragmentPath)
	if err != nil {
		fmt.Fprintf(environment.stdout, "%-10s not compared: cannot read %s (%v)\n",
			"settings", fragmentPath, err)
		return
	}

	drift, err := settings.Inspect(livePath, fragment)
	switch {
	case err != nil:
		fmt.Fprintf(environment.stdout, "%-10s not compared: %v\n", "settings", err)
	case drift.Absent:
		fmt.Fprintf(environment.stdout,
			"%-10s %s is absent; `petkit settings apply` would create it\n", "settings", shown)
	case drift.InStep():
		fmt.Fprintf(environment.stdout,
			"%-10s %s is in step with the fragment\n", "settings", shown)
	default:
		fmt.Fprintf(environment.stdout,
			"%-10s %s has drifted from the fragment in %d key(s); `petkit settings diff` says which\n",
			"settings", shown, len(drift.Changes))
	}
}

// statusOf is the whole of status once the manifest is in hand. setup reports
// through it rather than printing its own lines, so what setup says about a
// fresh clone is what status says about it a second later.
func statusOf(environment environment, loaded *manifest.Manifest) error {
	writer := tabwriter.NewWriter(environment.stdout, 0, 0, 2, ' ', 0)
	for _, itemState := range link.Inspect(loaded, environment.layout) {
		target := environment.layout.Display(itemState.Target)
		detail := ""
		switch itemState.Status {
		case link.Stale:
			detail = fmt.Sprintf("-> %s (want %s)",
				environment.layout.Display(itemState.Actual),
				environment.layout.Display(itemState.Source))
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

func commandSync(environment environment, dryRun bool) error {
	location, loaded, err := loadFrom(environment)
	if err != nil {
		return err
	}
	reportRepository(environment, location)
	return syncOf(environment, loaded, dryRun)
}

// syncOf is the whole of sync once the manifest is in hand — including the exit
// code a refusal is worth. `setup --sync` runs this, so it cannot install links
// by a path of its own that forgets to refuse something.
func syncOf(environment environment, loaded *manifest.Manifest, dryRun bool) error {
	actions := link.Plan(link.Inspect(loaded, environment.layout))

	prefix := ""
	if dryRun {
		prefix = "would "
	}
	writer := tabwriter.NewWriter(environment.stdout, 0, 0, 2, ' ', 0)
	for _, action := range actions {
		target := environment.layout.Display(action.State.Target)
		source := environment.layout.Display(action.State.Source)
		switch action.Kind {
		case link.Create:
			fmt.Fprintf(writer, "%screate\t%s\t%s -> %s\n", prefix, action.State.Item.ID, target, source)
		case link.Repoint:
			fmt.Fprintf(writer, "%srepoint\t%s\t%s -> %s (was %s)\n", prefix, action.State.Item.ID, target, source,
				environment.layout.Display(action.State.Actual))
		case link.None:
			fmt.Fprintf(writer, "ok\t%s\t%s\n", action.State.Item.ID, target)
		case link.Refuse:
			fmt.Fprintf(writer, "conflict\t%s\t%s (%s)\n", action.State.Item.ID, target, action.State.Detail)
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	result, err := link.Apply(actions, dryRun)
	if err != nil {
		return err
	}

	// ⚠️ A dry run closes with a sentence of its own rather than with nothing.
	// It used to print no closing line at all when it had something to do, while
	// printing one when it had nothing to do — so the quieter case got the
	// closing sentence and the louder one did not, and a reader who scrolls to
	// the bottom of a plan found the plan simply stopping. The `would ` prefix on
	// each item line said it, but only to someone reading every line. CLI-014.
	switch {
	case result.Changed == 0 && len(result.Refused) == 0:
		fmt.Fprintln(environment.stdout, "nothing to do; every item is already linked")
	case dryRun:
		fmt.Fprintf(environment.stdout, "%d change(s) would be made; nothing was changed\n", result.Changed)
	default:
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
	location, loaded, err := loadFrom(environment)
	if err != nil {
		return err
	}
	reportRepository(environment, location)

	findings := link.Doctor(loaded, environment.layout)
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

// withFragment resolves the repository and reads the settings fragment, which
// both settings subcommands need before they can say anything.
func withFragment(environment environment, action func(environment environment, livePath string, fragment []byte) error) error {
	location, err := locate(environment)
	if err != nil {
		return err
	}
	fragmentPath := filepath.Join(location.Root, "settings", "fragment.json")
	fragment, err := os.ReadFile(fragmentPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", fragmentPath, err)
	}
	livePath := filepath.Join(environment.layout.Config, "settings.json")
	return action(environment, livePath, fragment)
}

func settingsDiff(environment environment, livePath string, fragment []byte) error {
	// The same comparison `status` counts for its drift line. One reader, one
	// comparison: a second one would be free to disagree with the command the
	// drift line tells the reader to run.
	drift, err := settings.Inspect(livePath, fragment)
	if err != nil {
		return err
	}
	changes := drift.Changes
	shown := environment.layout.Display(livePath)
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
	shown := environment.layout.Display(livePath)
	if !result.Changed {
		fmt.Fprintf(environment.stdout, "%s already says what the fragment says; nothing was written\n", shown)
		return nil
	}
	if result.BackupPath != "" {
		fmt.Fprintf(environment.stdout, "backup %s\n", environment.layout.Display(result.BackupPath))
	}
	fmt.Fprintf(environment.stdout, "wrote  %s\n", shown)
	return nil
}

// pluginsManifestPath is the file `plugins plan` reads and `plugins capture`
// writes.
func pluginsManifestPath(root string) string {
	return filepath.Join(root, "settings", "plugins.json")
}

func commandPluginsPlan(environment environment) error {
	location, err := locate(environment)
	if err != nil {
		return err
	}
	desired, err := plugins.LoadDesired(pluginsManifestPath(location.Root))
	if err != nil {
		return err
	}
	live, err := plugins.LoadLive(filepath.Join(environment.layout.Config, "plugins"))
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

// commandPluginsCapture rewrites settings/plugins.json from what this machine
// actually has.
//
// ⚠️ It writes into the repository, not into the home directory, and it is the
// only command that does. That is why it goes through the same backup → temp →
// rename that `settings apply` uses: the file it replaces is hand-curated, and
// a capture that turns out to be wrong has to be recoverable without git.
func commandPluginsCapture(environment environment) error {
	location, err := locate(environment)
	if err != nil {
		return err
	}
	path := pluginsManifestPath(location.Root)

	captured, err := plugins.Capture(filepath.Join(environment.layout.Config, "plugins"))
	if err != nil {
		return err
	}
	content, err := plugins.Encode(captured)
	if err != nil {
		return err
	}

	// A missing file is the first capture, not a failure.
	var before *plugins.Desired
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if before, err = plugins.LoadDesired(path); err != nil {
			return err
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("cannot read %s: %w", path, err)
	}

	shown := environment.layout.Display(path)
	if bytes.Equal(existing, content) {
		fmt.Fprintf(environment.stdout,
			"%s already says what this machine has; nothing was written\n", shown)
		return nil
	}

	for _, line := range plugins.Changes(before, captured) {
		fmt.Fprintf(environment.stdout, "%s\n", line)
	}

	backupPath, err := settings.WriteWithBackup(path, content, 0o644, time.Now())
	if err != nil {
		return err
	}
	if backupPath != "" {
		fmt.Fprintf(environment.stdout, "backup %s\n", environment.layout.Display(backupPath))
	}
	fmt.Fprintf(environment.stdout, "wrote  %s\n", shown)
	fmt.Fprintln(environment.stdout,
		"\nThe version of each plugin is what this machine has today — a reading, not a pin.\n"+
			"Commit the file if this machine is the one the repository should describe.")
	return nil
}

func commandCheck(environment environment) error {
	location, err := locate(environment)
	if err != nil {
		return err
	}

	// environment.git, not state.GitRunner: git is a parameter for the same
	// reason the home directory is one, and check was the one command still
	// reaching for the real thing — which is why nothing could test it.
	result, err := state.Check(location.Root, environment.git)
	if err != nil {
		return err
	}

	// check printed its own `repository <absolute path>` line before CLI-010.
	// It prints the shared one now: the same question, answered the same way in
	// every command, and with the "how" it never used to say.
	reportRepository(environment, location)
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

// commandSetup turns a bare binary into a set-up machine: a clone, a recorded
// path, and a report of what `sync` would do.
//
// ⚠️ Without --sync it creates no symlink at all. Installing into `~/.claude` is
// `sync`'s decision to make and `sync`'s refusals to report, and a command whose
// job is "get me started" is the worst place to make it by surprise.
func commandSetup(environment environment, path string, withSync bool) error {
	result, err := state.Setup(state.SetupRequest{
		Path:       path,
		Home:       environment.layout.Home,
		ModulePath: environment.modulePath,
		Run:        environment.git,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(environment.stdout, "cloned   %s into %s\n", result.CloneURL,
		environment.layout.Display(result.Target))
	fmt.Fprintf(environment.stdout, "recorded %s in %s\n",
		environment.layout.Display(result.Location.Root),
		environment.layout.Display(result.Location.From))

	loaded, err := manifest.Load(result.Location.Root)
	if err != nil {
		return err
	}

	fmt.Fprintln(environment.stdout)
	if withSync {
		return syncOf(environment, loaded, false)
	}
	if err := statusOf(environment, loaded); err != nil {
		return err
	}
	fmt.Fprint(environment.stdout, `
setup made no links of its own. To install them:

  petkit sync
`)
	return nil
}

func commandInit(environment environment, path string) error {
	candidate := environment.workingDir
	if path != "" {
		candidate = path
	}
	location, err := state.Init(candidate, environment.layout.Home)
	if err != nil {
		return err
	}
	fmt.Fprintf(environment.stdout, "recorded %s in %s\n", location.Root, location.From)
	fmt.Fprintln(environment.stdout, "the other commands now work from any directory")
	return nil
}
