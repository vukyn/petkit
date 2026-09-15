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
| `CLI-001` | done | The command surface and the version rule |
| `CLI-003` | done | The command surface moved to `urfave/cli/v3` — every documented message byte-identical, and an undefined flag is now refused instead of silently ignored |
| `CLI-004` | done | `petkit version` printed a whole pseudo-version on a dirty tree: `+dirty` is build metadata and the anchored pattern did not allow for it |
| `CLI-005` | done | `petkit setup` clones the repository this binary was built from, records it, and reports what `sync` would do — install, one command, done |
| `CLI-006` | done | `CLAUDE_CONFIG_DIR` is honoured — petkit hard-coded `~/.claude`, so a machine that sets it was having everything installed where nothing reads it. Wrong on every OS, not only Windows |
| `CLI-007` | done | A missing git said `executable file not found in %PATH%`; it now names git and says what needs it |
| `CLI-002` | open | A binary can now clone itself a repository, but `check` still fails without one — what is left is whether it should answer from the tags API instead |
| `CLI-010` | done | The resolution was invisible: `status`, `sync`, `doctor` and `check` now open with which repository they resolved and how, and say both when the walked-up one is not the one `init` recorded |
| `CLI-011` | open | Every CLI test asserts with `strings.Contains` over the whole buffer, so a brand-new first line was invisible to all of them and a line going missing would be too |
| `CLI-012` | open | `make check` could not pass on Windows. The line endings are fixed — `.gitattributes` pins LF and `gofmt` is clean — but four tests still inject a platform while `path/filepath` and NTFS answer for the host |
| `CLI-009` | done | Run on a real Windows 11 machine at last. Every behaviour the port claimed held — an idempotent second `sync` above all — and the run found three things nothing on macOS could: `LINK-006`, `MFST-006` and `CLI-012` |
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
  process. ⚠️ It shipped compile-checked (`GOOS=windows` build and vet) and
  unit-checked **only**; it has since been run on a real Windows 11 machine
  (`CLI-009`) and every sentence above held, including the one that matters —
  a second `sync` says `nothing to do`. ⚠️ What the run found was three things
  no test on macOS could reach: `LINK-006` (a source missing at `sync` time
  makes a file-flavour symlink nothing repairs), `MFST-006` (`filepath.IsAbs`
  refuses `/x` as a source on macOS and accepts it on Windows) and `CLI-012`
  (**`make check` is red on Windows**, so the gate has never spoken there).

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

- [ ] `CLI-011` ⚠️ **Every CLI test asserts by searching the whole buffer, so
      the suite cannot see a line arrive or leave.** Raised 2026-09-14 out of
      `CLI-010`, which was expected to break these tests and did not.

      **What is wrong.** The assertions in `cmd/petkit/main_test.go` are
      `strings.Contains(stdout, "…")` over the entire output. That answers "does
      this phrase appear somewhere", which is a much weaker question than the one
      the tests are named for.

      **The measurement that raised it.** `CLI-010` added a **new first line** to
      `status`, `sync`, `doctor` and `check` — every command that resolves a
      repository. The brief predicted the byte-for-byte assertions would fail and
      that they would have to be updated. **Not one test moved**, and `make check`
      stayed green. A change to the first thing a user sees, on four commands, was
      invisible to 141 tests.

      ⚠️ **The symmetric case is the one that will bite.** A `Contains` assertion
      cannot notice a line *disappearing* either, as long as the phrase it hunts
      for survives elsewhere in the buffer. So the suite protects the words and
      not the shape: order, position, and whether a line exists at all are all
      unmeasured. The two tests `CLI-010` added read the first line by index
      (`strings.SplitN(stdout, "\n", 2)[0]`) precisely to sidestep this, and they
      are currently the only ones in the file that can.

      **What closing it needs.** Not a rewrite of every assertion — the phrase
      checks are fine for "this error names the item". What is missing is a
      **shape** assertion per command: the sequence of lines a command prints, in
      order, with the variable parts matched loosely. One helper, one table, and
      every command's skeleton becomes a thing the suite owns.

      ⚠️ **Do not close this by switching every `Contains` to an equality check.**
      Output that is fully pinned is output nobody can improve without a red
      suite, and this file already argues the opposite for messages that are the
      interface (`CLI-003`). The line to hold is: **the shape is asserted, the
      wording is sampled.**

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

- [ ] `CLI-012` ⚠️ **`make check` could not pass on Windows, for two reasons
      that have nothing to do with each other — and it had never been run there
      at all.** Raised 2026-09-15 by the Windows run (`CLI-009`). ⚠️ **Reason one
      shipped 2026-09-16; this entry now stands on reason two alone, and the gate
      is still red.**

      **Reason one — FIXED. gofmt failed on all 26 Go files, and the only
      difference was `\r`.** The repository carried no `.gitattributes`, so a
      checkout on a machine with `core.autocrlf=true` — the Git for Windows
      default — wrote every file CRLF. `gofmt` normalises to LF and therefore
      reported every file as unformatted. `gofmt -d internal/version/version.go`
      was 134 changed lines and **every one of them differed only by a trailing
      `\r`**. ⚠️ The gate was not wrong here; the checkout was. **What shipped:**
      `.gitattributes` carrying `* text=auto eol=lf`, and the working tree
      renormalised in a commit of its own — `gofmt -l .` has been empty on
      Windows since. ⚠️ That commit rewrote the line endings of every tracked
      file, which is why it was kept alone; a later change to that line owes the
      same treatment.

      **Reason two — OPEN. Four tests fail because the platform is a parameter
      but `path/filepath` is not.** They were six until `MFST-006` closed: two of
      those were a real defect in the product and are gone. These four are not.

      | test | says |
      | --- | --- |
      | `TestTildeIsTheOnlyTemplating` | `ExpandTilde("~") = "\tmp\home", want "/tmp/home"` |
      | `TestDisplayIsTheInverseOfExpansion` | `a path outside the home directory was collapsed to "\elsewhere\x"` |
      | `TestDisplayDoesNotFoldCaseOffWindows` | `a differently-cased Unix path was collapsed to "\home\ME\.claude\x"` |
      | `TestEverywhereElseALinkThatDiffersOnlyInCaseIsStale` | `status off Windows = "linked", want "stale"` |

      ⚠️ **This is the cost of the technique `CLI-009`'s port was built on, and
      it is worth stating plainly.** Passing the platform in as a parameter
      (`manifest.Layout.GOOS`, `internal/ospath`) makes a Windows branch runnable
      on macOS, and it was right to do. But `path/filepath` binds to `GOOS` at
      **compile** time and the host filesystem has its own opinion about case,
      so a test that injects "not Windows" while running on Windows is asking
      two authorities that disagree. The first three fail on the separator; the
      fourth fails because NTFS resolved a differently-cased path the injected
      platform was told to treat as a different file.

      ⚠️ **Two other failures were never this, and they are already gone.**
      `TestAnAbsoluteSourceIsRefused` and `TestAllProblemsAreReportedTogether`
      were `MFST-006`, a real defect in the product, fixed 2026-09-16. The four
      above are not defects in the product and **must not** be fixed by loosening
      an assertion.

      **What has to be decided, and it is a design question.** Either the
      cross-platform tests build their fixture paths through the injected
      platform instead of writing Unix literals — which means `ExpandTilde` and
      friends stop calling `filepath` directly and take the separator from
      `ospath` too — or those four tests are marked as running on one OS only,
      which gives up the property the technique was adopted for. ⚠️ **The second
      option is cheaper and worse**: it would leave the Windows branches
      unexercised on Windows, which is the state `CLI-009` existed to end.

      ⚠️ **`MFST-006` is a worked example of the first option and should be read
      before choosing.** It moved one question (`is this absolute?`) out of
      `path/filepath` and into `ospath` with the platform as an argument, and the
      test that resulted asserts both platforms' answers on whichever machine
      runs it. `ExpandTilde` is the same shape of problem, one size up.

      ⚠️ **Nothing may claim "`make check` is green on Windows" while this entry
      is open.** Half the reason is gone and the gate is still red; the honest
      sentence until reason two lands is "gofmt is clean and four tests fail".

      → `.gitattributes` (shipped), `Makefile` § `check`,
      `internal/manifest/manifest.go` (`ExpandTilde`), `internal/ospath`
      (`IsAbs` is the pattern), `internal/link/windows_test.go`,
      `internal/manifest/layout_test.go`.

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
