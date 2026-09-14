# TODO

A short index of what is done and what is not. It is deliberately **thin**:
nothing here explains a design, because the explanations live in `CLAUDE.md` (the
rules a change may not break), `README.md` (what the tool does) and `petkit.yaml`
(what is tracked, and what was rejected).

⚠️ **This file is hand-kept and nothing else here is.** The manifest is parsed,
the links are checked, the version is a git tag. A hand-kept list is the one thing
that can quietly become a lie.
**When you finish something, tick it here in the same commit.**

⚠️ **Refer to an entry by its CODE, never by its line number** — `LINK-002`,
`SETT-002`. A line number is a fact about how much prose sits above an item.

⚠️ **`## Not done` is open work only. Closing an item MOVES it**, in the same
commit as the work: a paragraph joins § *Done*, and the full record goes to
`docs/decisions.md` under the same code (the first item to close creates that
file). The check is one command and it must print nothing:

```sh
awk '/^## Not done/{f=1;next} /^## /{f=0} f && /^- \[x\]/' TODO.md
```

## The codes

`AREA-NNN`. **A code is permanent**, it never changes when an item is finished,
moved or refused, and a retired number is **never reused**. Anything new gets one
at the moment it is written down — a bug fixed in the same sitting included, as
`done` with its measurements. Take the next free number **by reading the index**,
not by counting entries.

⚠️ **The area is the SUBJECT, not the package the change lands in.** A settings
defect fixed in `internal/link` is still `SETT`.

| area | short for | what it covers |
|---|---|---|
| `LINK` | **link** | the symlink install itself: `sync`, `status`, `doctor`, and what may be removed |
| `MFST` | **m**ani**f**e**st** | `petkit.yaml`: schema, validation, and what is tracked |
| `SETT` | **sett**ings | `settings/fragment.json` and the merge into `~/.claude/settings.json` |
| `PLUG` | **plug**ins | `settings/plugins.json` and the `claude plugin` commands derived from it |
| `CLI` | **cli** | the command surface, versioning, `check`, `init`, and the gates |
| `DOC` | **doc**umentation | what this repository writes down about itself, and about the machine it manages |

**status** is `open` (§ *Not done*), `done` (§ *Done* carries the capability,
`docs/decisions.md` the record) or `refused` (§ *Decided against*).

| code | status | what it is |
|---|---|---|
| `LINK-001` | done | Symlink install: `status`, `sync`, `doctor` |
| `LINK-002` | open | A real directory where a link should go can only be refused — there is no way to adopt one |
| `LINK-004` | done | Windows refuses `os.Symlink` without Developer Mode or elevation; the errno is now translated into a message naming the cause and both ways out — and never into a copy |
| `LINK-005` | done | `status` called a good link `stale` on Windows, so `sync` removed and recreated it every run: path comparison now folds case on Windows only |
| `MFST-001` | done | The manifest and its validation |
| `MFST-004` | done | `~/` is a prefix, not a fence — `~/../elsewhere` passed validation and only `doctor` objected, after `sync` had already made the link. Moved into `validateTarget`; doctor's branch deleted as unreachable |
| `MFST-005` | done | A backslash in a target or source escaped the home directory on Windows and passed validation on macOS; refused everywhere, and the `/`→separator conversion is now explicit and watchable from here |
| `MFST-002` | open | `kind:` is decorative — nothing behaves differently per kind, so it is either a rule or a comment |
| `SETT-001` | done | The settings fragment and its merge |
| `SETT-002` | done | Nothing noticed settings drift until it was asked: `status` now carries one line saying whether the live file is in step with the fragment, drifted (and in how many keys), or absent |
| `PLUG-001` | done | The plugin list and `plugins plan` |
| `PLUG-004` | done | `petkit plugins capture` reproduces `settings/plugins.json` from the live files, reduced to `{id, scope, version}` plus each marketplace's source — and a test feeds it a `projectPath` and searches the written bytes for it |
| `PLUG-002` | done | `claude plugin install` has no version flag, so a version cannot be pinned: the field is relabelled **last seen** — a reading a machine can notice it is behind, not a target |
| `CLI-001` | done | The command surface and the version rule |
| `CLI-003` | done | The command surface moved to `urfave/cli/v3` — every documented message byte-identical, and an undefined flag is now refused instead of silently ignored |
| `CLI-004` | done | `petkit version` printed a whole pseudo-version on a dirty tree: `+dirty` is build metadata and the anchored pattern did not allow for it |
| `CLI-005` | done | `petkit setup` clones the repository this binary was built from, records it, and reports what `sync` would do — install, one command, done |
| `CLI-006` | done | `CLAUDE_CONFIG_DIR` is honoured — petkit hard-coded `~/.claude`, so a machine that sets it was having everything installed where nothing reads it. Wrong on every OS, not only Windows |
| `CLI-007` | done | A missing git said `executable file not found in %PATH%`; it now names git and says what needs it |
| `CLI-002` | open | A binary can now clone itself a repository, but `check` still fails without one — what is left is whether it should answer from the tags API instead |
| `CLI-010` | done | The resolution was invisible: `status`, `sync`, `doctor` and `check` now open with which repository they resolved and how, and say both when the walked-up one is not the one `init` recorded |
| `CLI-009` | open | Nothing has ever run on Windows — the port is compile-checked and unit-checked from macOS, and the machine itself is unmeasured |
| `DOC-001` | done | The fourteen stale copies were not shadowing anything — the plugin was installed project-scoped to another project and did not load here at all. Installed at user scope, the thirteen copies backed up and removed |
| `LINK-003` | refused | Installing by copy instead of by symlink |
| `CLI-008` | refused | Moving the config file to `os.UserConfigDir()` — it stays at `.config/petkit/config.json` on every OS |
| `PLUG-003` | refused | Installing plugins ourselves instead of printing `claude plugin` commands |
| `MFST-003` | refused | Tracking another repository's agents, commands and scripts |

## Done

- `LINK-001` **Symlink install.** `status` says `linked` / `missing` / `stale` /
  `conflict` per item; `sync` creates and repoints, `--dry-run` prints the same
  plan and touches nothing; `doctor` reports what is not about one item — a source
  that does not exist, a broken or unmanaged entry under the skills directory, a
  target outside the home directory. ⚠️ **The only thing `sync` may remove is a
  symlink**; anything else is a `conflict` and a non-zero exit.
- `MFST-001` **The manifest and its validation.** `petkit.yaml` lists items with
  an id, a kind, a source inside the repository and a target under `~`. Validation
  runs **before any action** — duplicate id, duplicate target, absolute target,
  source escaping the repository — so a half-applied `sync` is not reachable.
- `SETT-001` **The settings fragment.** `settings diff` and `settings apply` merge
  the declared keys of `settings/fragment.json` into `~/.claude/settings.json`,
  after a timestamped backup, through a temp file and a rename in the same
  directory. Keys the fragment does not name keep their values byte for byte.
- `PLUG-001` **The plugin list.** `settings/plugins.json` records the marketplaces
  and the plugins a machine should have; `plugins plan` prints the `claude plugin`
  commands that would close the gap and **never runs them**.
- `CLI-001` **The command surface.** Eight commands over `flag` and a switch, no
  framework. The version is the **git tag**, read from `debug.BuildInfo` — no
  constant, no `-ldflags`. The repository is found by walking up for
  `petkit.yaml`, then `$PETKIT_HOME`, then what `init` recorded.
- `CLI-005` **`petkit setup`.** Installing was three steps in two tools; it is
  now `go install` and `petkit setup`. Setup clones the repository **this binary
  was built from** — `debug.BuildInfo.Main.Path`, so a fork clones itself and
  there is no URL constant to get wrong — into `~/.petkit` or a path you name,
  records it through `petkit init`'s own recorder, and prints what `status`
  would print plus the next command. ⚠️ It **creates no symlink** without
  `--sync`, and it overwrites nothing: the target must be missing or empty.

- `PLUG-004` **`petkit plugins capture`.** `settings/plugins.json` is rebuilt
  from the live `~/.claude/plugins/{known_marketplaces.json,installed_plugins.json}`
  — through the `Layout`, so `CLAUDE_CONFIG_DIR` is honoured — reduced to
  `{id, scope, version}` per plugin plus each marketplace's source. ⚠️ **The
  reduction is enforced by the types**: the live-file structs do not declare
  `projectPath`, `installPath`, `installLocation`, `installedAt`, `lastUpdated`
  or `gitCommitSha`, so those are never read, and the test feeds a `projectPath`
  in and searches **the bytes that would be written** for it. It writes into the
  repository, so it goes through `settings apply`'s backup → temp → rename, and
  it prints what moved; a run that changes nothing writes nothing.

- `PLUG-002` **The recorded version is a reading, not a pin.** Measured
  2026-09-14: `claude plugin install --help` offers `--config`, `--json`,
  `-s/--scope`, `-y/--yes` and **no version flag at all**, so there is nothing to
  pin a version to. The field is therefore relabelled **last seen** — in
  `settings/README.md`, in the `Plugin` type, and in both lines of
  `plugins plan` that show a version — and kept, because a recorded reading is
  how a machine notices it is running something older. ⚠️ No pinning mechanism
  was invented and the field was not dropped.

- `SETT-002` **`status` notices settings drift.** One line at the end of
  `petkit status`: the live settings file is `in step with the fragment`,
  `has drifted … in N key(s)`, or `is absent`. It is `settings diff`'s own
  comparison (`settings.Inspect`, which both commands call), not a second one —
  a second comparison would be free to disagree with the command the line tells
  you to run. ⚠️ `status` still exits **0** on every path through it, including
  the paths where the comparison cannot be made at all.

- `CLI-010` **The resolved repository is visible.** `status`, `sync`, `doctor`
  and `check` open with one line — always — saying which repository they resolved
  and how: `walked up from the working directory`, `$PETKIT_HOME`, or
  `recorded by petkit init`. ⚠️ When the walked-up repository is **not** the one
  `init` recorded, the line says both, because that is the case where `sync`
  repoints every link into a clone nobody chose. Not applied to `version` (it
  already names the manifest path) or `setup` (it prints the clone it just made).
  ⚠️ The resolution **order** is unchanged on purpose: making the recorded path
  win would break a fresh checkout being usable before `init`.

- `LINK-004` / `LINK-005` / `MFST-005` / `CLI-006` / `CLI-007` **Windows.** The
  tool is now written to be correct on Windows: the symlink privilege refusal is
  translated into a message naming Developer Mode and an Administrator shell
  (and ⚠️ **never** into a copy — one copy is the whole design); path comparison
  folds case on Windows only, so a good link is no longer called `stale` and
  repointed every run; a backslash in a manifest path is refused everywhere,
  because it escaped the home directory on Windows while passing validation on
  macOS; `CLAUDE_CONFIG_DIR` decides where `~/.claude` is on **every** OS; and a
  missing `git` is named. ⚠️ **No `//go:build windows` file was added.** The
  platform is a parameter — `manifest.Layout.GOOS`, and every function in the new
  `internal/ospath` — so a macOS test runs the Windows branch in the same
  process. ⚠️ It is compile-checked (`GOOS=windows` build and vet) and
  unit-checked; **nothing has run on Windows** — that is `CLI-009`.

## Not done

- [ ] `LINK-002` **A real directory where a link should go can only be refused.**
      Raised 2026-09-14 with the first version.

      **What is missing.** The first `sync` on a machine that already has
      `~/.claude/skills/writing-todo` as a real directory reports a `conflict` and
      stops. The way out is manual: look at it, move it, run `sync` again. That is
      the right default and the right first release — ⚠️ **the alternative, a tool
      that moves a directory it did not create, is the one failure this repository
      cannot afford**, because the targets are inside `~/.claude`.

      **What closing it needs.** Either an explicit `--adopt` that moves the
      existing directory to `~/.claude/.petkit-adopted/<id>-<timestamp>/` and
      *then* links, or a decision that the manual path is the whole answer and this
      entry becomes a refusal. Whichever, the narrow case must keep its test: a
      conflict with no flag stays refused, with the directory's contents intact.

- [ ] `MFST-002` **`kind:` is decorative.** Raised 2026-09-14 with the first
      version.

      Every item carries `kind: skill` and **nothing reads it** — the link logic is
      the same for a file and a directory. So it is either a rule (a `kind` decides
      the default target directory, or which validation applies) or it is a comment
      pretending to be a field. ⚠️ A field that is parsed and ignored is worse than
      no field: the next person adds `kind: script` and reasonably expects it to
      change something.

- [ ] `CLI-002` **`check` still needs a clone.** Raised 2026-09-14 with the first
      version, **measured the same day**; the premise moved under it when
      `CLI-005` shipped.

      `go install github.com/vukyn/petkit/cmd/petkit@latest` works — measured
      against a clean module cache with no `GOPRIVATE` and no credentials, both
      `@latest` and `@v0.1.0` install and the binary reports `petkit v0.1.0`.

      **What `CLI-005` took off this entry.** "The binary cannot get itself a
      clone" is no longer true: `petkit setup` clones and records in one command,
      and `check` answers normally from that moment on — measured end to end, a
      fresh binary against a disposable `HOME` reports `checkout on v0.2.0` a
      second after setup returns. The install story is closed; this entry is not.

      **What is left.** `check`, `sync`, `status` and `doctor` on a machine with
      no repository *and* no setup still get:

      ```
      petkit: cannot find the petkit repository: no petkit.yaml above /tmp,
      PETKIT_HOME is not set, and nothing is recorded in
      ~/.config/petkit/config.json — run `petkit init /path/to/petkit` once, or
      run petkit from inside the repository
      ```

      ⚠️ Two things that message does not yet say, and they are the open work:
      it names `petkit init` but not `petkit setup`, which is now the answer for
      a machine that has no clone **at all** — and the original question is
      untouched: should `check` answer "is there a newer tag" **without** a
      clone, from the GitHub tags API? It is the one question a machine can
      sensibly ask before it has cloned anything, and setup did not answer it
      because setup's answer is "clone first".

      **What closing it needs.** Either `check` falls back to the GitHub tags API
      when there is no repository, or its error names both ways out in one
      sentence — which is cheap and might be the whole answer.

- [ ] `CLI-009` ⚠️ **Nothing has ever run on Windows.** Raised 2026-09-14 with
      the port that made the code correct there.

      **What is measured.** `GOOS=windows GOARCH=amd64 go build ./...` and
      `GOOS=windows go vet ./...` are clean, and every Windows branch has a unit
      test that runs on macOS with the platform passed in as a parameter. That is
      the whole of the evidence. **Compiling is not running**, and no line of
      this has touched a Windows filesystem, a Windows `%USERPROFILE%`, a Windows
      `git`, or a real `ERROR_PRIVILEGE_NOT_HELD`.

      **What to run, in the order the risk runs.** ⚠️ Do this in a shell with
      Developer Mode **off** first — the interesting failure is the one that does
      not happen on a developer's own box.

      1. `go install github.com/vukyn/petkit/cmd/petkit@latest`, then
         `petkit setup`. This is the first thing a new machine does and the first
         thing that runs `git`. If git is missing you should get the sentence
         naming git, not `executable file not found in %PATH%`.
      2. `petkit sync` **without** Developer Mode. Expect the
         `ERROR_PRIVILEGE_NOT_HELD` message naming Developer Mode and
         Administrator. ⚠️ If it instead reports `conflict`, or succeeds, the
         errno did not arrive in the shape the translation expects — dump
         `%+v` of the raw error before changing anything.
      3. Turn Developer Mode on, `petkit sync`, then **`petkit sync` again**.
         The second run must print `ok` for every item and `nothing to do`. ⚠️
         **This is the single most important line to look at.** If it prints
         `repoint`, the case/separator comparison is still wrong and petkit is
         deleting and recreating a good symlink on every run.
      4. `petkit status` and `petkit doctor`. Targets should print as
         `~/.claude/skills/x` with forward slashes; doctor must not list an item
         petkit itself installed as "not in the manifest".
      5. `set CLAUDE_CONFIG_DIR=%USERPROFILE%\somewhere-else` and repeat 3 and 4.
      6. `petkit check` in the clone.

      **Two things known to be unmeasurable from macOS, so look at them first.**

      - ⚠️ **`doctor`'s survey under a Windows layout.** The decision
        underneath it — does this directory entry belong to an item? — is tested
        directly in `doctor_internal_test.go`, because a Windows layout resolved
        on a macOS filesystem produces paths this filesystem cannot hold in the
        shape Windows would (the remainder's separator becomes a backslash, and
        `~/.claude/skills` is then not that file's parent). The **wiring** from
        `os.ReadDir` into that decision is therefore unexercised. Step 4 above is
        the test.
      - ⚠️ **A missing source makes a broken link, not a refused one.** Go's
        `os.Symlink` on Windows picks the file flavour or the directory flavour
        of a symlink by looking at the destination, so a source that is not there
        yet yields a *file* symlink pointing at a directory — broken in a way
        Unix does not reproduce, because Unix symlinks have no flavour. `doctor`
        reports a missing source; `sync` does not refuse one. Reproduce by
        removing a source directory and running `sync`.

      **One defect deliberately left open.** Two items claiming `~/x` and `~/X`
      are one target on Windows and two everywhere else. Validation does **not**
      fold case, because a manifest valid on macOS and invalid on Windows is the
      machine-specific `petkit.yaml` the whole design refuses. On Windows those
      two items will fight over one path, run after run. If it ever happens, the
      answer is a refusal in `validateTarget` that is case-insensitive on every
      platform — not one that consults `GOOS`.

## Decided against — do not re-raise

- `LINK-003` **Installing by copy instead of by symlink.** Considered on
  2026-09-14 and rejected for the property that decides it: with a copy, a skill
  edited on the machine that wrote it is **invisible** until somebody remembers to
  copy it back, and the two versions drift silently in the meantime. With a
  symlink, editing the installed skill *is* editing the repository, and
  `git status` sees it the same second. The cost is real and accepted: deleting
  the clone breaks every link, and a mistake made in `~/.claude/skills` is a
  mistake made in the repository. ⚠️ Re-raise only for a machine that cannot keep
  a clone — and then as a *second mode*, not as a replacement.

- `CLI-008` **Moving the config file to `os.UserConfigDir()`.** Considered on
  2026-09-14 while making the tool correct on Windows, and refused: it would put
  the record in `%AppData%` on Windows, `~/Library/Application Support` on macOS
  and `~/.config` on Linux — three paths for one file. The path is part of the
  interface. It is quoted in `petkit version`'s "not read" note, in the "cannot
  find the petkit repository" message, in `README.md` and in what `petkit init`
  prints, and one path means an instruction written on one machine is correct on
  another. It stays `.config/petkit/config.json` everywhere. ⚠️ The full record,
  including why `~/.claude` moved and this did not, is in `docs/decisions.md`
  under `CLI-008` — re-raise only from there.

- `PLUG-003` **Installing plugins ourselves instead of printing `claude plugin`
  commands.** `claude plugin` already resolves marketplaces, caches by commit sha,
  and records scope in `installed_plugins.json`; reimplementing that means owning a
  second source of truth for what is installed, and the two would disagree the
  first time somebody used the real one. What this repository adds is the **list**
  — the thing `claude plugin` does not keep across machines. ⚠️ Re-raise only with
  a concrete failure of the printed-commands flow, not with "it would be nicer".

- `MFST-003` **Tracking another repository's agents, commands and scripts.**
  Eight agents, five commands and two scripts were considered and left where they
  are: each is addressed *relative to the repository that owns it* —
  `$CLAUDE_PROJECT_DIR/scripts/<name>.sh` in a hook, a `$(SCRIPTS)` variable in a
  Makefile — and arrives on every machine with `git clone`. A copy here would be a
  second version of a tracked file with no reader. ⚠️ Re-raise only for a piece
  that stops being repository-scoped.
