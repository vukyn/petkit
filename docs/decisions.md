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
