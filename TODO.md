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
| `MCP` | **mcp** | `settings/mcp.json` and the `claude mcp add -s user` commands derived from it — user-scope servers only |
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
| `LINK-006` | open | A source that is missing when `sync` runs yields a Windows **file**-flavour symlink to a directory: a directory scan cannot see it and `sync` never repairs it. `doctor` at least stopped saying the destination does not exist when it does |
| `MFST-001` | done | The manifest and its validation |
| `MFST-004` | done | `~/` is a prefix, not a fence — `~/../elsewhere` passed validation and only `doctor` objected, after `sync` had already made the link. Moved into `validateTarget`; doctor's branch deleted as unreachable |
| `MFST-005` | done | A backslash in a target or source escaped the home directory on Windows and passed validation on macOS; refused everywhere, and the `/`→separator conversion is now explicit and watchable from here |
| `MFST-006` | done | The absolute-source refusal asked `filepath.IsAbs`, so `/x` was refused on macOS and accepted on Windows: `ospath.IsAbsAnywhere` now asks both platforms, and a rule about a travelling file stops consulting the machine it runs on |
| `MFST-002` | open | `kind:` is decorative — nothing behaves differently per kind, so it is either a rule or a comment |
| `SETT-001` | done | The settings fragment and its merge |
| `SETT-002` | done | Nothing noticed settings drift until it was asked: `status` now carries one line saying whether the live file is in step with the fragment, drifted (and in how many keys), or absent |
| `PLUG-001` | done | The plugin list and `plugins plan` |
| `PLUG-004` | done | `petkit plugins capture` reproduces `settings/plugins.json` from the live files, reduced to `{id, scope, version}` plus each marketplace's source — and a test feeds it a `projectPath` and searches the written bytes for it |
| `PLUG-002` | done | `claude plugin install` has no version flag, so a version cannot be pinned: the field is relabelled **last seen** — a reading a machine can notice it is behind, not a target |
| `PLUG-005` | done | `plugins plan` read the enabled set out of `installed_plugins.json`, which `claude plugin install` rewrites without it, and printed `claude plugin enable` for a plugin `claude` called already enabled: it now reads `settings.json`'s `enabledPlugins`, the legacy key only as a fallback |

| `MCP-001` | done | The user-scope MCP server list and `mcp plan` |
| `MCP-002` | done | `petkit mcp capture` reproduces `settings/mcp.json` from `~/.claude.json`'s `mcpServers` key, reducing every env var and header to its NAME — and a test feeds it a planted secret value and searches the written bytes for it |
| `CLI-001` | done | The command surface and the version rule |
| `CLI-003` | done | The command surface moved to `urfave/cli/v3` — every documented message byte-identical, and an undefined flag is now refused instead of silently ignored |
| `CLI-004` | done | `petkit version` printed a whole pseudo-version on a dirty tree: `+dirty` is build metadata and the anchored pattern did not allow for it |
| `CLI-005` | done | `petkit setup` clones the repository this binary was built from, records it, and reports what `sync` would do — install, one command, done |
| `CLI-006` | done | `CLAUDE_CONFIG_DIR` is honoured — petkit hard-coded `~/.claude`, so a machine that sets it was having everything installed where nothing reads it. Wrong on every OS, not only Windows |
| `CLI-007` | done | A missing git said `executable file not found in %PATH%`; it now names git and says what needs it |
| `CLI-002` | done | Twice the premise moved and twice the answer was smaller than the title: `CLI-005` gave a binary its own clone, and what was actually wrong was one sentence pointing a new machine at `petkit init`. The tags-API half is refused as `CLI-013` |
| `CLI-010` | done | The resolution was invisible: `status`, `sync`, `doctor` and `check` now open with which repository they resolved and how, and say both when the walked-up one is not the one `init` recorded |
| `CLI-011` | done | The suite protected the words and not the shape: four lines were deleted from production one at a time and two of them were invisible to all 141 tests. One helper, one table, nine command shapes -- each line in order, count asserted, wording still sampled |
| `CLI-014` | done | `sync --dry-run` printed no closing line when it had something to do and one when it had nothing to do: it now closes with "N change(s) would be made; nothing was changed", and two tests that used `Contains("change(s)")` as a proxy now assert the last line they meant |
| `CLI-012` | done | `make check` could not pass on Windows and now exits 0 there: `.gitattributes` pins LF, `ospath.Clean`/`Join` take the platform as an argument so three fixtures can ask both, and the fourth test skips on a measured case-folding filesystem — a question no volume that folds case can be asked |
| `CLI-009` | done | Run on a real Windows 11 machine at last. Every behaviour the port claimed held — an idempotent second `sync` above all — and the run found three things nothing on macOS could: `LINK-006`, `MFST-006` and `CLI-012` |
| `DOC-001` | done | The fourteen stale copies were not shadowing anything — the plugin was installed project-scoped to another project and did not load here at all. Installed at user scope, the thirteen copies backed up and removed |
| `LINK-003` | refused | Installing by copy instead of by symlink |
| `CLI-008` | refused | Moving the config file to `os.UserConfigDir()` — it stays at `.config/petkit/config.json` on every OS |
| `PLUG-003` | refused | Installing plugins ourselves instead of printing `claude plugin` commands |
| `MFST-003` | refused | Tracking another repository's agents, commands and scripts |
| `CLI-013` | refused | `check` answering "is there a newer tag" from the GitHub tags API with no clone — four failure modes bought for a question `petkit setup` disposes of in one command |
| `MCP-003` | refused | Tracking project- and local-scope MCP servers |

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

- `PLUG-005` **The enabled set comes from `settings.json`.** Measured
  2026-11-03 on Windows: after `claude plugin install`, `installed_plugins.json`
  (version 2) held no `enabledPlugins`, `plugins plan` printed
  `claude plugin enable context-mode@context-mode`, and `claude` refused it as
  already enabled — the answer sits in `~/.claude/settings.json`. `LoadLive` and
  `Capture` now take the config directory and read `enabledPlugins` there; the
  key in `installed_plugins.json` is read only when `settings.json` has none.
  Mutation: reading the legacy key first turns four tests red.

- `MCP-001` **The user-scope MCP server list.** `settings/mcp.json` records the
  `claude mcp add -s user` servers a machine should have; `mcp plan` prints the
  commands that would close the gap and **never runs them**. ⚠️ **Only user
  scope is tracked** — see `MCP-003`. A server's `command` (and, for `http`/
  `sse`, its `url`) is kept as the absolute path or address a live machine
  recorded, because `claude mcp add` cannot start it without one; `mcp plan`
  attaches an advisory note next to it instead of stripping it, naming it
  machine-specific. There is **no version concept**: measured 2026-11-03,
  `claude mcp add --help` has no version-pinning flag at all, so `Server` has
  no `Version` field — present-and-empty or otherwise.

- `MCP-002` **`petkit mcp capture`.** `settings/mcp.json` is rebuilt from the
  live `~/.claude.json`'s `mcpServers` key — measured 2026-11-03 by diffing
  `~/.claude.json` before and after `claude mcp add -s user ...`, once for a
  stdio server and once for an `http` one, then removing both — reduced to
  `{name, transport, command, args, url}` per server plus the **NAMES**
  (never the values) of its env vars and headers, sorted. ⚠️ **The value
  reduction is enforced by discipline, not by the type the way `PLUG-004`'s
  is**: `liveServer`'s `Env`/`Headers` fields do hold the live values (there
  is no field-shaped way to read a map's keys without also being able to read
  its values), so `Capture` is the one place in the package that is trusted
  to read them and only ever calls the key-listing helper on them. The test
  plants a fake secret as a named constant and searches **the bytes that
  would be written** for it, the same discipline `PLUG-004`'s test holds
  `plugins.Capture` to. ⚠️ **A secret typed directly into `args` is invisible
  to this reduction and stays in the file** — a documented limitation, not a
  gap nobody noticed; see `settings/README.md`.

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
  process. ⚠️ It shipped compile-checked (`GOOS=windows` build and vet) and
  unit-checked **only**; it has since been run on a real Windows 11 machine
  (`CLI-009`) and every sentence above held, including the one that matters —
  a second `sync` says `nothing to do`. ⚠️ What the run found was three things
  no test on macOS could reach: `LINK-006` (a source missing at `sync` time
  makes a file-flavour symlink nothing repairs), `MFST-006` (`filepath.IsAbs`
  refuses `/x` as a source on macOS and accepts it on Windows) and `CLI-012`
  (**`make check` could not pass on Windows at all**, so the gate had never
  spoken there). `MFST-006` and `CLI-012` are closed and `make check` has
  exited 0 on Windows since 2026-09-16; `LINK-006` is still open on the half
  that would repair a wrong-flavour link rather than describe one.

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

- [ ] `LINK-006` ⚠️ **A source that is missing when `sync` runs leaves a Windows
      *file*-flavour symlink pointing at a directory, nothing ever repairs it,
      and the skill then disappears from every consumer that scans the
      directory.** Raised 2026-09-15 by the Windows run (`CLI-009`), which
      predicted the shape and then reproduced it.

      ⚠️ **Option 1 below shipped 2026-09-16: `doctor` no longer makes a false
      statement about the filesystem. Everything else in this entry still
      stands** — the link is still the wrong flavour, `sync` still calls it `ok`,
      and the skill is still invisible to a directory scan. The reading improved;
      the machine did not.

      **What is wrong.** Go's `os.Symlink` on Windows picks the file flavour or
      the directory flavour by looking at the destination. A source that is not
      there yet is not a directory, so the link is created with
      `FILE_ATTRIBUTE_ARCHIVE` where a good one gets `FILE_ATTRIBUTE_DIRECTORY`.
      ⚠️ Unix does not reproduce it at all — a Unix symlink has no flavour — and
      `sync` does not refuse a missing source: it exits 0 and makes the link.

      **The evidence.** A throwaway repository with two items, one source present
      and one absent, synced on Windows 11 Pro 26200 with `CLAUDE_CONFIG_DIR`
      pointed at a scratch directory:

      | item | source when `sync` ran | attributes after | flavour |
      | --- | --- | --- | --- |
      | `skill/present` | present | `Directory, ReparsePoint` | directory — right |
      | `skill/absent` | absent | `Archive, ReparsePoint` | **file** — wrong |

      The missing source was then created and everything re-run:

      | command | says | true? |
      | --- | --- | --- |
      | `petkit sync` | `ok`, `nothing to do; every item is already linked` | nothing was repaired |
      | `petkit status` | `linked` | — |
      | `petkit doctor`, **before 2026-09-16** | `is a broken symlink: it points at …, which does not exist` | **no** — it exists; and doctor exits 0, so the false sentence stayed a note run after run |
      | `petkit doctor`, **now** | `is a symlink petkit cannot follow: it points at …, and following it failed: Access is denied.` | yes — measured on the same fixture |

      ⚠️ **`doctor`'s old sentence was false, and one missing check was why.**
      `brokenSymlink` in `internal/link/doctor.go` reasoned that "a symlink that
      Lstats but does not Stat is exactly this case". On Windows the `os.Stat`
      of a file-flavour link to a directory fails with **`Access is denied.`**,
      and `os.IsNotExist(err)` is **`false`** — so a permission error was
      reported as a missing destination.

      **What shipped for it (option 1).** `followSymlink` returns one of four
      states instead of a bool, and only `os.IsNotExist` produces "does not
      exist"; anything else says the link could not be followed and quotes why.
      ⚠️ **`os.Stat` is a parameter**, because the error Windows produces cannot
      be staged on a Unix filesystem — the same move `internal/ospath` makes for
      the platform, and the reason the branch can be watched failing at all.
      Mutation: dropping the `IsNotExist` branch turns the new unit test **and**
      the existing `TestDoctorReportsABrokenSymlinkUnderTheSkillsDirectory` red.
      ⚠️ The finding is still `Info` and doctor still exits 0 on it — raising
      that is part of option 2 or 3, not this.

      **What it costs, measured against the real consumer.** Reading a known path
      *through* the link works; **scanning the directory does not**:

      | runtime | call | result |
      | --- | --- | --- |
      | Go | `os.Stat(link)` | `Access is denied.`; `os.IsNotExist` = `false` |
      | Go | `os.ReadDir(link)` | `Access is denied.` |
      | Node | `fs.statSync(link).isDirectory()` | `EPERM: operation not permitted, stat` |
      | Node | `fs.readdirSync(link)` | `EPERM: operation not permitted, scandir` |
      | Node | `fs.readFileSync(link + '/SKILL.md')` | succeeds |
      | PowerShell | `Test-Path -PathType Container` / `-PathType Leaf` | `False` / `True` |
      | PowerShell | `Get-ChildItem`, `Get-Content` | both succeed |

      ⚠️ **Claude Code discovers skills by scanning, and Node cannot scan this
      link.** The failure is therefore not cosmetic: the skill is silently absent
      from the one thing petkit exists to configure, while `petkit status` says
      `linked` and `petkit sync` says `nothing to do`.

      ⚠️ **This is not `LINK-002`.** That one is a **real** directory where a
      link should go, which `sync` refuses and leaves alone. This is a link
      petkit itself created and calls `ok`.

      **Three places could own it; deciding which is what is left.**
      1. ~~`doctor` stops reporting every Stat failure as "does not exist".~~
         **SHIPPED 2026-09-16** — the only one that merely stopped a false
         sentence: no behaviour changed and nothing needed deciding.
      2. `sync` refuses an item whose source is not there. Widens what `sync`
         **refuses**, which is not what `CLAUDE.md` § *The rule the whole tool
         rests on* guards — read it anyway before touching the neighbours.
      3. `status` / `sync` call a wrong-flavour link `stale` and repoint it. ⚠️
         **This widens what `sync` may DELETE**, to "a symlink whose flavour
         disagrees with its destination". It therefore owes a test proving a real
         file and a real directory are still refused, and a line here saying why
         the rule moved.

      ⚠️ **2 and 3 are not alternatives to each other in the obvious way.** A
      `sync` that refuses a missing source (2) stops the bad link being made
      tomorrow; it does nothing about one made yesterday, which is what every
      machine that already ran the old `sync` is carrying. Only 3 repairs those,
      and 3 is the one that touches the rule this repository rests on.

      ⚠️ **Reproduce on Windows, not in a unit test.** The flavour is chosen by
      the Windows kernel, not by any branch petkit owns, so the platform-as-a-
      parameter technique cannot reach it. What option 1 showed is the next best
      move: the *error* the kernel produces can be a parameter even when the
      kernel cannot.

      → `internal/link/doctor.go` (`followSymlink`, done), `internal/link/link.go`
      (`sync`'s side, open), `docs/decisions.md` § `CLI-009`.

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

- `CLI-013` **`check` answering "is there a newer tag" from the GitHub tags API,
  with no clone.** Considered on 2026-09-16 out of `CLI-002`, and rejected on what
  it costs against what it buys. The idea is sound, which is why it stayed open
  for two days: a binary knows its own version (`debug.BuildInfo`), so "am I
  behind?" is genuinely answerable with no repository at all, and it is the one
  question a machine can ask before it has cloned anything. ⚠️ **What decides it
  is the other side of the ledger.** It puts a network call, a GitHub dependency,
  an offline path and a rate limit into a tool that has none of those and is
  deliberately local, small and dependency-shy — four new failure modes, each
  owing a message and a test — to answer a question `petkit setup` disposes of in
  one command, by producing the clone that makes `check` work properly rather
  than approximately. ⚠️ **Re-raise only for a machine that must know it is
  behind and may not clone**: a locked-down image, or a CI job that installs the
  binary alone. And then as a **second command** (`petkit latest`), never as a
  fallback inside `check` — a `check` that answers from a different source
  depending on whether a directory happens to exist is two commands wearing one
  name, and the reader cannot tell which one answered.

- `MCP-003` **Tracking project- and local-scope MCP servers.** Considered and
  refused alongside `MCP-001`/`MCP-002`, 2026-11-03. `claude mcp add` has three
  scopes: `user` (this repository's whole concern), `project` (written into
  the project's own `.mcp.json`, committed with that project) and `local`
  (tied to one machine AND one project path at once). Project-scope is the
  same shape `MFST-003` already decided: it is addressed *relative to the
  repository that owns it* and arrives with `git clone`, so a copy here would
  be a second version of a tracked file with no reader. Local-scope cannot be
  refused the same way — there is no other repository for it to belong to —
  it is refused because a single cross-machine record cannot represent it
  honestly: "this machine, in this one checkout" has no "which machine, which
  checkout" field to put the other half of the pair in, and a record that
  silently assumed one machine and one checkout would be wrong everywhere
  else it was read. ⚠️ Re-raise only for a concrete need to describe a single
  project-local MCP server across machines, and then as a record that
  carries both halves explicitly rather than pretending one global list can
  mean the same thing on every clone.
