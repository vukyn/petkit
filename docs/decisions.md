# Decisions

Every piece of work that is **finished**, with the reasoning and the measurements
that decided it. `TODO.md` says what is open; this says why a finished thing is
the way it is. The entries carry the same `AREA-NNN` codes.

- [x] `MFST-004` ⚠️ **`~/` is a prefix, not a fence — and the rule that caught an
      escaping target was in the wrong place.** Found and fixed 2026-09-14, while
      reviewing the first version before it was committed.

      **What was wrong.** A target had to start with `~/`, and `~/../elsewhere/x`
      does. Expanded, it is `filepath.Join(home, "../elsewhere/x")` — the parent of
      the home directory. Validation accepted it; only `doctor` objected, and
      `doctor` is a command somebody runs **after** `sync` has already created the
      link. The rule existed and was enforced one step too late.

      **What shipped.** `validateTarget` refuses a target whose cleaned remainder
      climbs out (`..` or `../…`), naming the item and quoting the target. Two
      tests: the refusal, and its positive case — `~/.claude/skills/../skills/x`
      **is** accepted, so the guard refuses the escape rather than the two dots.
      Mutation-checked: deleting the four-line guard turns
      `TestATargetThatClimbsOutOfHomeIsRefused` red.

      ⚠️ **`doctor`'s own outside-home branch was deleted, not kept as a
      backstop.** Once validation refuses the shape, no manifest that reaches
      `Doctor` can carry it, so the branch could not fire — and a guard that cannot
      fire is not defence in depth, it is a line that reads like protection. Its
      test went with it, and the reason is written where the branch was.
      `manifest.UnderHome`, the exported helper it used, had no other caller and
      was removed too.

      **The tell, for next time.** The brief said "a target must start with `~/`"
      and the implementation said exactly that. The gap was between two rules that
      each looked complete: validation owned the *shape* and doctor owned the
      *location*, and nothing owned "a shape that produces a bad location". When
      two checks split one property, ask which of them runs before the action.

- [x] `CLI-003` **The command surface moved to `urfave/cli/v3`.** Done 2026-09-14,
      at the owner's request, to match the house style of the other CLIs in the
      surrounding workspace.

      **What shipped.** `github.com/urfave/cli/v3 v3.11.0`. The command graph
      lives in `newRootCommand`; `run` keeps its signature and still owns the exit
      codes (2 usage, 1 failed, 0 fine). Help and version are rendered through
      `cli.HelpPrinter` and `cli.VersionPrinter`, so the framework prints what
      petkit already printed. **Nothing under `internal/` changed** — verified with
      `git diff --name-only -- internal/`, which is empty: two wrappers in
      `cmd/petkit` absorbed the signature the framework wanted.

      **The proof that the messages did not move.** Both binaries were built with
      `-buildvcs=false` and run back to back over thirteen invocations — every
      command, `help` in three spellings, an unknown command, no arguments at all,
      and each subcommand group with and without a valid subcommand — comparing
      stdout, stderr **and** exit code. The diff is empty. A second comparison ran
      `sync` end to end against a disposable `HOME`.

      ⚠️ **The first comparison showed two phantom differences and neither was a
      regression**: Go's VCS stamp put `+dirty` in the version string, and
      `go get` had already dirtied `go.mod` so `check`'s dirty count moved. A
      before/after diff is only evidence when the only difference between the two
      binaries is the code — `-buildvcs=false` is what made it one.

      **One message did change, deliberately.** An undefined flag used to print
      the `flag` package's stock usage dump; it now prints
      `petkit sync: flag provided but not defined: -bogus`. The old text was never
      petkit's — reproducing it would mean hand-maintaining a second copy of the
      flag list inside the port that deletes it. Two consequences, both on inputs
      that were previously undefined: `petkit status --foo` was **exit 0 with the
      flag silently ignored** and is now exit 2, and `--help` works on every
      subcommand where it used to be a parse error.

      **The help test is the one that had to be built right.** It walks
      `root.VisibleCommands()` rather than matching a literal string, so a command
      dropped during a future refactor takes the help line with it and the test
      fails. Mutation-checked: removing `check` from the graph fails it three
      times, once per spelling of help.

- [x] `CLI-004` ⚠️ **`petkit version` printed a whole pseudo-version on a dirty
      tree.** Found 2026-09-14 while verifying `CLI-003`, fixed the same day.

      **What was wrong.** A build from a working tree with uncommitted changes
      carries a `+dirty` build-metadata suffix:
      `v0.1.1-0.20260914063411-8d81ea5ae0fd+dirty`. `pseudoVersion` was anchored
      with `$` and made no room for it, so the match failed and `Format` fell
      through to "leave a tag whole" — printing fourteen digits of timestamp where
      a revision was meant.

      **What shipped.** The pattern allows an optional `+…` suffix and the suffix
      is **kept** in the output: `petkit version` now prints
      `petkit 8d81ea5ae0fd+dirty`. "Which commit" and "plus uncommitted changes"
      are two different answers and a reader building from a local checkout needs
      both. Mutation-checked: dropping the optional group turns
      `TestADirtyPseudoVersionIsStillTrimmedToItsRevision` red, and the positive
      sibling holds that a real tag is still left whole with or without metadata.

      ⚠️ **Invisible from a clean tree and from a `go install`ed binary**, which
      is why it survived the release. The version rule has three shapes — a tag, a
      pseudo-version, and a pseudo-version from a dirty tree — and the third only
      exists while somebody is working.

- [x] `CLI-005` **`petkit setup` — installing is one command now.** Done
      2026-09-14, out of `CLI-002`'s measurement that a binary with no clone can
      only answer `version`.

      **What shipped.** `petkit setup [path] [--sync]`, default path `~/.petkit`.
      It derives the repository from the module this binary was built from,
      refuses a target that is not missing-or-empty, clones, records the path
      through **`petkit init`'s own recorder** (`state.Init`, not a second
      writer), and then prints what `status` would print for the fresh manifest
      plus the exact next command. `--sync` runs the same code `petkit sync`
      runs and carries its exit code. Three steps in two tools became
      `go install` + `petkit setup`.

      **The URL is not a constant, and that is the point.** It is
      `"https://" + debug.BuildInfo.Main.Path` — the same build info the version
      comes from. A fork installed from its own module path clones **itself**; a
      constant would send it to this repository, and the failure would look like
      success: a working clone of the wrong person's setup. ⚠️ The test that
      holds the line asserts **the argument the fake runner received**, not
      anything printed, because a constant smuggled back in prints an equally
      plausible line. Mutation-checked: replacing the derivation with
      `"https://github.com/vukyn/petkit"` turns
      `TestTheCloneURLIsBuiltFromTheModulePath` red.

      **What setup deliberately does not do.**

      - ⚠️ **It creates no symlink without `--sync`.** Writing into `~/.claude`
        is `sync`'s decision, with `sync`'s conflict rule and `sync`'s exit code,
        and a command whose job is "get me started" is the worst place to make it
        by surprise. The test snapshots the home directory either side of the
        command and allows exactly one addition — the record — and no symlink at
        all. Mutation-checked: making setup always sync turns
        `TestSetupWithoutSyncMakesNoLinks` red.
      - **It removes nothing.** The target must be missing or an empty
        directory; a directory with anything in it, or a file, is named and left.
        This is `sync`'s "only a symlink may be removed" rule applied to the one
        directory setup writes, and the test **reads the file back out** rather
        than stat'ing it. Mutation-checked, as is the refusal of a machine that
        already has a recorded, resolving repository — a second clone of the
        thing whose whole point is that there is one.
      - **It does not clone shallow.** No `--depth`: `petkit check` compares
        tags, and a shallow clone arrives without them. Asserted in the same test
        that asserts the URL, and confirmed on the real clone
        (`rev-parse --is-shallow-repository` → `false`, `tag` → `v0.2.0`).

      **Measured end to end**, binary built with `go build`, against a disposable
      `HOME` and a scratch directory: the clone happened, the record landed in
      `~/.config/petkit/config.json`, `status` reported both items `missing`,
      and `petkit check` — run from a directory with no `petkit.yaml` above it —
      answered `checkout on v0.2.0`. `--sync` into a second disposable `HOME`
      created both links; over a pre-existing real directory it created the other
      one, refused that one by name, exited 1, and the file in the way still read
      `somebody else's`.

      ⚠️ **Two things about the derivation that do not hold, written down because
      they will not be obvious later.**

      1. **"Build info is unavailable" is almost never the failing shape.**
         `go run` carries a module path (measured: `ok=true`,
         `path="example.com/…"`, `version="(devel)"`), and so does a plain
         `go build`. The case that actually produces an empty path is a build
         outside module mode — `GO111MODULE=off` in GOPATH — where
         `debug.ReadBuildInfo()` still returns `ok=true` and `Main.Path` is `""`.
         The guard is therefore on the **empty path**, not on the `ok` flag, and
         the message says "this build carries no module path" rather than naming
         `go run`.
      2. **A `/vN` module suffix would break it.** `github.com/vukyn/petkit/v2`
         is a valid module path and `https://github.com/vukyn/petkit/v2` is not a
         repository. petkit has no `/v2` and may never have one, so nothing
         trims it — but if this repository ever takes a major version, `CloneURL`
         is the line that has to learn about it, and the symptom will be a clone
         that 404s rather than anything subtle.

- [x] `LINK-004` ⚠️ **A symlink needs a privilege on Windows, and the refusal
      arrived as a raw syscall number.** Fixed 2026-09-14, in the port that made
      the tool correct on Windows.

      **What was wrong.** `os.Symlink` fails with `ERROR_PRIVILEGE_NOT_HELD`
      (1314) unless Developer Mode is on or the process is elevated. `sync`
      wrapped it as `cannot link X -> Y for item "z": A required privilege is not
      held by the client`, which names neither cause nor cure. It is the first
      thing a new Windows machine hits, and the message sends the reader nowhere.

      **What shipped.** `link.ExplainSymlinkFailure` translates that one errno
      into a sentence carrying the item id, the target path, the constant's name
      (`ERROR_PRIVILEGE_NOT_HELD` — the string the reader will search for), and
      both ways out: Developer Mode in Settings, or an Administrator shell.
      `linkFailure` is the single place a symlink failure becomes a message, so
      the create path and the repoint path cannot diverge. Every other error
      keeps its wording byte for byte.

      ⚠️ **It does not fall back to copying, and that is the decision, not an
      omission.** A copy on the machine that cannot link would leave the
      installed skill and the repository free to drift, which is exactly what
      `LINK-003` refused and what the symlink design exists to prevent. The
      message says so in as many words, so the next reader does not add the
      fallback as a kindness.

      **How it is measured without Windows.** The errno is a plain
      `syscall.Errno(1314)` constant, not behind a build tag, so a macOS test
      builds the `*os.LinkError` Windows would produce and watches the
      translation. No Unix errno comes near 1314 — Linux stops in the 130s,
      darwin in the 100s — so the match cannot fire by accident on the platform
      that cannot produce it. Two tests: the translation, and the positive
      sibling proving an ordinary error (`no space left on device`) and a nil
      error both pass through untranslated. Mutation-checked: forcing the guard
      always-true turns `TestTheWindowsSymlinkPrivilegeFailureIsExplained` red.

- [x] `LINK-005` ⚠️ **`status` called a perfectly good link `stale` on Windows,
      and `sync` then performed the one destructive operation petkit has.** Fixed
      2026-09-14.

      **What was wrong.** `samePath` compared a symlink's destination with the
      item's source using `==`. On Windows `C:\x` and `c:\x` are one file, and
      the two spellings reach petkit from two places — one out of `os.Readlink`,
      one built from the manifest. A byte comparison reports `stale`; `Plan`
      turns `stale` into `Repoint`; `Repoint` **removes the symlink and recreates
      it**. Not a cosmetic mislabel: the only removal petkit is ever allowed to
      make, fired for no reason, on every single run.

      **What shipped.** A new package, `internal/ospath`, holding the path
      questions whose answer depends on the platform — `Equal`, `Key`, `Under`,
      `FromSlash`, `ToSlash` — each taking the platform as an **argument**.
      `samePath`, `Layout.Display` and doctor's managed-set all compare through
      it. On Windows the comparison folds case and accepts either separator;
      everywhere else it is the byte comparison it was.

      ⚠️ **macOS is deliberately not folded**, even though its default filesystem
      also ignores case. Whether an APFS volume folds is a property of how it was
      formatted, so folding there would be a guess — and the generous direction
      of that guess makes petkit call a genuinely stale link `linked`, which is
      the wrong way to be wrong. Windows is the only special case.

      ⚠️ **Manifest validation was deliberately NOT made case-insensitive.** Two
      items claiming `~/x` and `~/X` are one target on Windows and two
      everywhere else, but folding in `validateTarget` would make a manifest
      valid on macOS and invalid on Windows — and the one thing `petkit.yaml` may
      not be is machine-specific. It stays a run-time confusion rather than a
      portable refusal; see the open Windows entry.

      **Tests.** `TestOnWindowsALinkThatDiffersOnlyInCaseIsLeftAlone` asserts both
      the status **and** the plan, because the plan is where the harm is; its
      sibling `TestEverywhereElseALinkThatDiffersOnlyInCaseIsStale` asserts the
      opposite verdict and a `Repoint`. The fixture is built so it reads the same
      on a case-sensitive volume and a case-insensitive one. Mutation-checked:
      restoring `==` in `samePath` turns the Windows test red.

- [x] `MFST-005` ⚠️ **A manifest path written with a backslash escaped the home
      directory on Windows and passed validation on macOS.** Found and fixed
      2026-09-14, while auditing where a `/`-written target becomes a real path.

      **What was wrong.** `MFST-004` refuses `~/../elsewhere` by cleaning the
      remainder and looking for `..` or `../`. On macOS `filepath.Clean` leaves
      `..\elsewhere` exactly as it is — a backslash is an ordinary character in a
      Unix file name — so the check saw one harmless component and accepted it.
      On Windows the same string cleans to `..\elsewhere` **as two components**
      and resolves to the parent of the home directory. The manifest travels
      between machines, so the machine that wrote it passed a check the machine
      that read it needed. `source:` had the identical hole out of the
      repository.

      **What shipped.** A backslash in a target or a source is refused outright,
      on every platform, naming the item and quoting the value. One spelling,
      checked once, meaning the same thing everywhere — which is the same
      argument that made `~` the only templating.

      **The rest of the audit.** `filepath.Join` and `filepath.Clean` already
      convert `/` on Windows, so the joins were not broken; what was missing was
      that the conversion could not be *watched* from here. `ospath.FromSlash`
      now performs it explicitly with the platform as a parameter, at the two
      boundaries where manifest text becomes a path — `Layout.Resolve` and
      `Item.SourcePath` — and `ospath.ToSlash` performs the inverse at the one
      boundary where a path becomes text, `Layout.Display`. A path printed back
      is now spelled the way `petkit.yaml` spells it (`~/.claude/skills/x`) on
      every platform, so the line in `petkit status` and the line in the manifest
      are the same string. `doctor`'s survey no longer splices a `/`-written
      constant at all: `SkillsDirName` is gone, replaced by `Layout.SkillsDir()`.

      **Tests.** `TestAManifestTargetResolvesUnderAWindowsRoot` and
      `TestASourceResolvesUnderAWindowsRoot` resolve a `/`-written manifest path
      under `C:\Users\me`, with the expectation written as a join onto a
      remainder spelled with a **backslash** — an assertion that is correct on
      both hosts and fails here the moment the conversion is dropped.
      `TestTheSameTargetResolvesNativelyOnThisMachine` is the sibling.
      `TestAPathWrittenWithBackslashesIsRefused` covers target, source and the
      escaping case; `TestTheSamePathsWrittenWithSlashesAreAccepted` proves the
      rule refuses the backslash and not the path. All mutation-checked.

- [x] `CLI-006` ⚠️ **petkit installed into a directory nothing reads on any
      machine that sets `CLAUDE_CONFIG_DIR`.** Fixed 2026-09-14.

      **What was wrong.** Claude Code honours `CLAUDE_CONFIG_DIR`; petkit
      hard-coded `~/.claude` in four places — the manifest targets, the settings
      merge, the plugin survey and doctor's skills directory. On a machine that
      sets the variable, `sync` reported success, `status` reported `linked`, and
      Claude Code never saw any of it. ⚠️ **Not a Windows defect** — it is wrong
      the same way on macOS and Linux, and it was found while auditing for
      Windows only because both questions are "where does this path actually
      come from".

      **What shipped.** `manifest.Layout` — the machine a manifest is applied to:
      what `~` means, what `~/.claude` means, and which platform's path rules
      apply. `NewLayout` reads the variable (it wins when set and non-empty, and
      a `~` inside it expands like any other), and every command takes the layout
      rather than a bare home directory. `manifest.CollapseHome` was replaced by
      `Layout.Display` and removed: two ways to spell one path is how the four
      hard-codings happened in the first place.

      ⚠️ **Only the configuration directory moves.** A target that is not under
      `~/.claude` is still relative to the home directory — the variable says
      where Claude Code's configuration lives, not where the user does. Asserted.

      ⚠️ **The layout carries GOOS as a field**, set from `ospath.Current()` in
      `main` and from a literal in a test. That is what lets a macOS test resolve
      and print paths the way Windows would, in the same process, with no build
      tag — and it is why this port added no `//go:build windows` file at all.

      **Tests.** Measured end to end rather than at the resolver, because the
      hard-codings were in the commands: `TestSyncInstallsIntoClaudeConfigDir`
      (the link lands in the configured directory **and** nothing is left in
      `~/.claude`), `TestDoctorSurveysTheClaudeConfigDir`,
      `TestSettingsAndPluginsFollowClaudeConfigDir`, plus the unit cases for the
      variable's precedence, its `~` expansion, and the unset default. Every one
      mutation-checked.

- [x] `CLI-007` **A missing git said `executable file not found in %PATH%`.**
      Fixed 2026-09-14.

      `setup` clones and `check` compares tags, both through `git`. Without git
      on `PATH`, `exec` answers with a sentence that reads like an internal error
      and never says that installing git is the fix. It is the likeliest first
      failure on a fresh Windows machine, where git is not part of the system the
      way it is in a developer's Unix shell.

      `state.DescribeGitFailure` names the case in one sentence, saying which two
      commands need git and what to do. Every other git failure keeps git's own
      words byte for byte — `git describe --tags in /repo: fatal: …` is
      unchanged, and so is the fallback to the exit error when stderr is empty.
      Three tests, the first built from `&exec.Error{Err: exec.ErrNotFound}`
      because this machine has git; mutation-checked.

- [x] `CLI-008` **The config file stays at `.config/petkit/config.json` on every
      OS.** Decided 2026-09-14, while making the tool correct on Windows.

      `os.UserConfigDir()` would put it in `%AppData%` on Windows,
      `~/Library/Application Support` on macOS and `~/.config` on Linux — three
      paths for one file. It is **refused**, and the reason is that the path is
      part of the interface: it is quoted in `petkit version`'s "not read" note,
      in the "cannot find the petkit repository" message, in `README.md`, and in
      the output of `petkit init`. One path means one answer to "where is the
      record" — in a message, in a README, in a support question — and it means
      an instruction written on one machine is correct on another.

      ⚠️ **This is the record.** A convention doc will one day say to use
      `os.UserConfigDir()`; it was considered here and rejected on purpose, and
      `state.ConfigRelPath` is deliberately a single `/`-written constant run
      through `filepath.FromSlash`. `~/.claude` is a separate question and got a
      separate answer — it moved, because `CLAUDE_CONFIG_DIR` is another
      program's decision that petkit has to follow (`CLI-006`). The config file
      is petkit's own, and petkit keeps it in one place.

- [x] `DOC-001` ⚠️ **The fourteen stale skill copies were not shadowing anything.
      The plugin was not loaded at all.** Raised 2026-09-14 as "it is unmeasured
      whether they shadow the plugin's own", closed the same day — and the premise
      was wrong in a way worth keeping.

      **What the question assumed.** That a personal skill directory and a plugin
      skill of the same name compete, and one wins. The real answer is that there
      was no competition: the plugin was installed with **`scope: project`, bound
      to a different project**, so in this workspace it contributed nothing.
      `enabledPlugins` listed only one plugin, and it was not this one.

      ⚠️ **The measurement that settled it was not a config file.** Configuration
      says what should happen; what settled it is that `using-git-worktrees` exists
      **only** in the plugin and was **absent from the session's own skill list**.
      A skill the plugin provides and the machine does not have is proof the plugin
      is not loaded — the same shape of evidence as running a guard and watching it
      fire, rather than reading the line that says it would.

      **How stale the copies were.** Measured against the installed 6.3.0 cache:

      | skill | plugin adds | local-only lines | files only in the plugin |
      | --- | ---: | ---: | ---: |
      | `subagent-driven-development` | +447 | −156 | 4 |
      | `using-superpowers` | +132 | −148 | 3 |
      | `writing-skills` | +64 | −152 | 6 |
      | `systematic-debugging` | +7 | −20 | **10** |
      | `verification-before-completion` | +0 | −19 | 0 |

      ⚠️ **The local-only lines were not edits.** A scan of every such line across
      all thirteen for personal markers — repository names, an employer's name, any
      Vietnamese diacritic — found **nothing**, and a sampled diff showed upstream
      prose that 6.3.0 had deleted. So they were an older release carried forward,
      not customisation, and removing them cost nothing.

      **What shipped.** `claude plugin install superpowers@claude-plugins-official
      --scope user`; the thirteen copies moved to
      `~/.claude/.superpowers-replaced-<timestamp>/` rather than deleted. Four
      entries remain in the skills directory: two petkit symlinks and two symlinks
      into another tool's directory. `settings/plugins.json` records the new scope.

      ⚠️ **6.3.0 was already the newest** — the local marketplace pin, the upstream
      marketplace pin and `obra/superpowers`'s own manifest all agree on
      `b36e0829` / `6.3.0`. The update that was needed was not a version bump; it
      was a **scope**. "Is it the latest" and "is it loaded" are different
      questions, and only the second one was wrong.

- [x] `PLUG-004` ⚠️ **`settings/plugins.json` was reduced by hand, and the
      obvious rebuild leaks an absolute path.** Closed 2026-09-14.

      **What was wrong.** The file records `{id, scope, version}` per plugin. Its
      source, `~/.claude/plugins/installed_plugins.json`, records more: an
      `installPath` on every entry, an `installedAt`, a `gitCommitSha`, and on
      every project-scoped entry a `projectPath` — an absolute path into whatever
      repository the plugin was installed for. `known_marketplaces.json` carries
      an `installLocation` the same way. The committed file was clean **because
      somebody reduced it by hand**, and nothing in the repository performed or
      enforced that reduction. ⚠️ This matters whether or not the repository is
      public: a private repository is still shared with whoever is added to it,
      and a path is the kind of thing that is pasted into an issue without
      thought.

      **What shipped.** `petkit plugins capture`. It reads both live files
      through `Layout` (so `CLAUDE_CONFIG_DIR` is honoured, `CLI-006`), writes
      exactly the id, scope and version per plugin plus each marketplace's
      `{repo, source}`, prints what moved, and — because it writes into the
      **repository**, the only file petkit rewrites outside the home directory —
      goes through `settings apply`'s own backup → temp → rename. That path is
      now `settings.WriteWithBackup`, called by both commands: a second copy of
      it is a second answer to "was it backed up", and the copy that gets it
      wrong is the one nobody watched being written. A run that finds nothing
      changed prints so and writes nothing, which is why `Encode` is
      deterministic (plugins sorted by id, maps sorted by `encoding/json`) and
      why a test captures the same machine twenty times and compares bytes —
      **map iteration order is random per range, so one comparison would agree by
      chance half the time.**

      ⚠️ **The reduction is performed by the types, not by remembering.**
      `liveMarketplace` and `liveInstalled` do not declare the fields that must
      not travel, so they are never read, and `Desired` has nowhere to put them
      if they were. That is why the mutation that proves the test needed **two**
      edits — declare `projectPath` on the live entry *and* pass it through into
      a captured field. Declaring it alone changes nothing, which is the property
      the design was after.

      **The test is the point, and it searches the bytes.**
      `TestACaptureCarriesNoPathOutOfTheLiveFiles` feeds a live pair containing
      a `projectPath`, an `installPath` and an `installLocation`, encodes the
      capture, and asserts
      the written bytes contain none of them — nor the field names, nor
      `/Users/`. Its positive sibling asserts the four things that **must** be
      there, so it is measuring the reduction and not a capture that writes
      nothing. ⚠️ A capture whose reduction can only be confirmed by reading the
      code is the same hand-reduction with more steps.

      **One thing the brief did not name, and it is kept.** The file also has an
      `enabled` map, which `plugins plan` reads to print `claude plugin
      enable|disable`. It is `id -> bool`, carries no path, and dropping it would
      make the first real capture **delete** a key the committed file has — a
      capture that cannot reproduce the file it replaces is not a capture. It is
      written, and `Desired`'s field order was changed to
      `enabled, marketplaces, plugins` so the bytes match the file's existing key
      order.

- [x] `PLUG-002` **The recorded plugin version is a reading, not a pin.**
      Measured and closed 2026-09-14.

      **The measurement that decided it.** `claude plugin install --help` offers
      `--config`, `--json`, `-s/--scope`, `-y/--yes` and **no version flag at
      all**. There is nothing to pin a version to, so `plugins plan` printing
      `claude plugin install <id>` is not an oversight — it is the only thing the
      command can be told. At least one marketplace entry also carries
      `autoUpdate: true`, which would move a plugin underneath a pin even if one
      existed.

      **What shipped: a label, not a mechanism.** The field means **last seen**,
      and says so in `settings/README.md`, on the `Plugin` type, and in both
      lines of `plugins plan` that show a version — the reason on an install
      command ("last seen at X … a reading, not a pin, so this installs whatever
      the marketplace offers today") and the note on a version difference ("a
      difference to look at rather than one to correct").

      ⚠️ **The field was not dropped**, and that is the decision. A recorded
      reading is how a machine notices it is running something older; without it
      `plugins plan` can only say "installed" or "not installed", and two
      machines on different versions look identical. ⚠️ **No pinning mechanism
      was invented.** The test holds both halves: no `claude plugin install` line
      may contain a version (mutation: appending `@%s` to it turns the test red),
      and every line that shows one must call it a reading (mutation: restoring
      the old "the manifest captured version %s" wording turns it red).

- [x] `SETT-002` **Nothing noticed when a machine's `settings.json` drifted.**
      Closed 2026-09-14.

      **What was wrong.** `settings diff` answers the question when asked, and
      nothing asked it. A machine where the fragment was applied months ago and
      `skillOverrides` has since been edited by hand reports `linked` on every
      skill and is still not the machine the repository describes. ⚠️ It is not a
      symlink problem: the settings file is genuinely merged rather than linked,
      because it holds machine-local keys the repository must not own.

      **What shipped.** One line at the end of `petkit status`: the live file is
      `in step with the fragment`, `has drifted from the fragment in N key(s);
      petkit settings diff says which`, or `is absent`.

      ⚠️ **It is `settings diff`'s own comparison, not a second one.**
      `settings.Inspect` reads the live file (treating a missing one as the
      `Absent` case rather than an error) and returns `Diff`'s answer; `status`
      counts it and `settings diff` prints it. A second comparison written for
      the one-line answer would be free to disagree with the very command the
      line tells the reader to run.

      ⚠️ **`status` still exits 0 on every path**, including the two where the
      comparison cannot be made at all — no fragment in the repository, or a live
      file that is not JSON. Those print `settings   not compared: …`. Drift is a
      fact about a machine, not a broken command, and a `status` that can fail is
      a `status` nobody can put in a script.

      **Tests.** `TestStatusSaysWhetherTheSettingsFileIsInStep` walks one sandbox
      through all three states — absent, drifted in exactly two keys, then in
      step a moment after `settings apply` — asserting exit 0 each time.
      Mutation-checked twice: replacing the change count with `0` turns it red,
      and reporting every file as in step turns it red.

- [x] `CLI-010` ⚠️ **The resolved repository was invisible, so running petkit
      inside a second clone silently repointed every link into it.** Closed
      2026-09-14.

      **What was wrong.** The repository is resolved by walking up from the
      working directory first, then `$PETKIT_HOME`, then the path `init`
      recorded. A machine set up at one path, whose shell happens to sit inside
      another clone, therefore gets the other one — and `sync` repoints every
      symlink into it. Nothing was wrong from petkit's side, so nothing said
      anything; the only way to notice was to read the paths in the plan.

      **What shipped: the cheap option, deliberately.** `status`, `sync`,
      `doctor` and `check` open with one line saying which repository was
      resolved and how — `walked up from the working directory`, `$PETKIT_HOME`,
      or `recorded by petkit init` — **always**, not only when something looks
      odd. ⚠️ And when the walked-up repository is **not** the one `init`
      recorded, the line names both: `repository <used> (walked up …) — but
      petkit init recorded <other>`. That case is the actual defect; a line that
      only printed the winner would have been decoration.

      ⚠️ **The resolution order is unchanged.** Making the recorded path win
      would break a fresh checkout being usable before `init` has ever run, which
      is the case the walk-up exists for. Asking before repointing was the other
      option and was not taken: a prompt is a second thing to get right in a
      command that runs unattended after a `git pull`, and it does not help the
      three commands that only read.

      **Where the answer is decided.** `state.Location` gained `How` (one of
      three `state.Source` constants, so four commands cannot word it three ways)
      and `Recorded`, which is filled in whenever something is recorded and the
      root did not come from it. ⚠️ `Recorded` is filled in **even when it
      agrees**: whether two paths name one file is a question about a platform
      (`C:\x` and `c:\x` are one file on Windows), and `internal/state` is not
      told which platform it is on. The printer is — it holds the `Layout` — so
      the printer compares, through `ospath.Equal`. The read of the config file
      on those two paths swallows every error on purpose: they resolve a
      repository **without** the record today and must not start failing because
      of a file they did not need.

      **Which messages moved.** `petkit check` printed
      `repository <absolute path>` as its first line; it prints the shared line
      instead, which spells the path the way every other petkit message spells it
      (`~/…`) and adds the `(how)`. `status`, `sync` and `doctor` gained a first
      line where they had none, and `status` a last one (`SETT-002`).
      `petkit version` and `petkit setup` were left alone: version already names
      the manifest path it read, and setup prints the clone it just made.
      ⚠️ **No existing test had to be rewritten** — this repository's CLI tests
      assert by `strings.Contains` rather than against a whole captured output,
      so the new first line was invisible to all of them. That is worth knowing
      for the next change of this shape: **nothing here would have caught a line
      going missing**, which is why the new tests read the first line
      specifically rather than searching the whole buffer.

      **Tests.** `TestEveryCommandSaysWhichRepositoryItResolvedAndHow` checks the
      **first** line of `status`, `sync --dry-run` and `doctor`, then the other
      two resolution sources one at a time (mutation: printing a constant `how`
      turns it red).
      `TestAWalkedUpRepositoryThatIsNotTheRecordedOneNamesBoth` sets up a second
      clone, records it, runs from inside the first, and asserts both paths
      appear; its positive sibling runs the same command from a neutral working
      directory and asserts the second half is **not** printed (mutation:
      comparing the root with itself turns it red).
