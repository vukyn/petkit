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
| `MFST-001` | done | The manifest and its validation |
| `MFST-004` | done | `~/` is a prefix, not a fence — `~/../elsewhere` passed validation and only `doctor` objected, after `sync` had already made the link. Moved into `validateTarget`; doctor's branch deleted as unreachable |
| `MFST-002` | open | `kind:` is decorative — nothing behaves differently per kind, so it is either a rule or a comment |
| `SETT-001` | done | The settings fragment and its merge |
| `SETT-002` | open | Nothing notices when a machine's `settings.json` drifts from the fragment |
| `PLUG-001` | done | The plugin list and `plugins plan` |
| `PLUG-002` | open | Two machines can end up on different plugin versions and the manifest cannot say otherwise |
| `CLI-001` | done | The command surface and the version rule |
| `CLI-002` | open | `check` needs a clone; a machine that only ran `go install` has no repository to check against |
| `DOC-001` | open | Fourteen stale skill copies still sit in `~/.claude/skills`, and it is unmeasured whether they shadow the plugin's own |
| `LINK-003` | refused | Installing by copy instead of by symlink |
| `PLUG-003` | refused | Installing plugins ourselves instead of printing `claude plugin` commands |
| `MFST-003` | refused | Tracking pet-platform's agents, commands and scripts |

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

- [ ] `SETT-002` **Nothing notices when a machine's `settings.json` drifts from
      the fragment.** Raised 2026-09-14 with the first version.

      `settings diff` answers the question when asked, and nothing asks it. A
      machine where the fragment was applied months ago and `skillOverrides` has
      since been edited by hand reports `linked` on every skill and is still not
      the machine the repository describes. ⚠️ This is **not** a symlink problem —
      the settings file is genuinely merged, not linked, because it holds
      machine-local keys the repository must not own.

      **What closing it needs.** A drift line in `status` (the shared exit path),
      or a decision that `settings diff` on demand is enough — with the reason.

- [ ] `PLUG-002` **Two machines can end up on different plugin versions and the
      manifest cannot say otherwise.** Raised 2026-09-14 with the first version.

      `settings/plugins.json` captured what this machine had on 2026-09-13,
      versions included (`context-mode 1.0.169`, `ui-ux-pro-max 2.13.0`,
      `caveman 15581d14007f` — a commit sha, not a version). But `plugins plan`
      prints `claude plugin install <id>`, which takes whatever the marketplace
      offers today, and `caveman` has `autoUpdate: true` in its marketplace entry.
      So the recorded version is a **reading, not a pin**.

      ⚠️ Do not "fix" this by pinning without checking that `claude plugin install`
      accepts a version at all — if it does not, the honest close is to relabel the
      field as *last seen* and stop implying it is a target.

- [ ] `CLI-002` **`check` needs a clone.** Raised 2026-09-14 with the first
      version.

      The advertised install is `go install …@latest` plus a `git clone`, because
      the symlink targets have to point at *something* — so a clone always exists
      in the intended flow. But `petkit check` shells out to `git` in that clone,
      and a machine that installed the binary and never cloned gets an error where
      it expected a version answer. ⚠️ `petkit version` works everywhere;
      `check` is the one command with a hidden requirement.

      **What closing it needs.** Either `check` falls back to the GitHub tags API
      when there is no repository, or its error says exactly that in one sentence
      — which is cheap and might be the whole answer.

- [ ] `DOC-001` ⚠️ **Fourteen stale skill copies still sit in `~/.claude/skills`,
      and it is unmeasured whether they shadow the plugin's own.** Raised
      2026-09-14 out of the curation that produced this repository.

      **What is known.** `~/.claude/skills` holds fourteen directories that are
      older copies of skills the installed `superpowers` plugin also provides.
      Measured 2026-09-13: the plugin's `writing-plans/SKILL.md` carries a
      *Task Right-Sizing* section the local copy lacks, and its `brainstorming` is
      92K against the local 64K. Six of the fourteen are `off` in
      `settings/fragment.json`, which means eight are live.

      ⚠️ **What is NOT known is which copy wins** when a personal skill directory
      and a plugin skill share a name. If the local one shadows, then nine skills
      are silently running an old version on this machine, and that is a defect
      with a fix (delete them) rather than untidiness. If the plugin wins, they are
      dead weight and the fix is the same but the urgency is not.

      **What closing it needs.** One reading — invoke a shadowed skill and check
      which text arrives (the *Task Right-Sizing* section is a one-word probe) —
      then delete the fourteen, or record why they stay. ⚠️ `petkit` must not do
      the deleting: they are not its items, and a tool that removes things it does
      not manage is exactly what `LINK-002` refuses.

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

- `PLUG-003` **Installing plugins ourselves instead of printing `claude plugin`
  commands.** `claude plugin` already resolves marketplaces, caches by commit sha,
  and records scope in `installed_plugins.json`; reimplementing that means owning a
  second source of truth for what is installed, and the two would disagree the
  first time somebody used the real one. What this repository adds is the **list**
  — the thing `claude plugin` does not keep across machines. ⚠️ Re-raise only with
  a concrete failure of the printed-commands flow, not with "it would be nicer".

- `MFST-003` **Tracking pet-platform's agents, commands and scripts.** Eight
  agents, five commands and two scripts were considered and left where they are:
  they are addressed *relative to the repository that owns them* —
  `$CLAUDE_PROJECT_DIR/scripts/security-scan-check.sh` in a hook,
  `$(SCRIPTS)/hosts.sh` in the Makefile — and arrive on every machine with
  `git clone`. A copy here would be a second version of a tracked file with no
  reader. ⚠️ Re-raise only for a piece that stops being repository-scoped.
