# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What this is

`petkit` keeps one person's Claude Code setup — skills, a settings fragment, a
plugin list — in git, and installs it into `~/.claude` by **symlink**. It is a
standalone CLI: no domains, no DI, no web framework, no database, no UI. Whatever
service conventions the surrounding workspace carries do not apply here.

`TODO.md` is the open list, and every item in it carries a permanent code
(`AREA-NNN`). Name an item by its **code** in a commit, a PR or a note, never by
a line number or a title.

## The rule the whole tool rests on

**Nothing destructive without proof of ownership.** `sync` may remove exactly one
kind of thing: a symlink. A regular file or a real directory where a target
should go is a `conflict` — named, left alone, exit non-zero.

⚠️ This is not defensive politeness. The targets are inside `~/.claude`, which
holds work nobody has a second copy of, and the tool runs unattended enough
(`petkit sync` after a `git pull`) that a wrong deletion would be discovered long
after the fact. **Any change that widens what `sync` may delete needs a test that
proves the narrow case still refuses**, and an entry in `TODO.md` saying why the
rule moved.

## Invariants

- **Version is the git tag.** `debug.BuildInfo.Main.Version`, read at run time.
  No version constant, no `-ldflags -X`, nothing to keep in step. A tag is left
  whole; a pseudo-version is trimmed to its revision.
- **`~` is the only templating** a manifest target may use. An absolute target is
  refused: it makes the file machine-specific, which is the one thing `petkit.yaml`
  may not be. `$HOME` and other variables are not expanded — one rule, testable.
- **Manifest validation runs before any action**, not during it. A duplicate id, a
  duplicate target, a source that escapes the repository: all refused up front, so
  a half-applied `sync` is not a state the tool can reach.
- **Every write to a user's file** goes backup → temp in the same directory →
  `os.Rename`. Same directory matters: a rename across filesystems is not atomic.
- **`settings apply` touches declared keys only.** The fragment is a slice, not a
  replacement. A key it does not name keeps its value byte for byte, and there is
  a test that puts an unrelated nested object in the live file and asserts it
  survives.
- **`plugins plan` prints and never executes.** `claude plugin` owns installation;
  this repository owns the list. `plugins capture` is the one writer of that list,
  and it writes **only** `{id, scope, version}` per plugin plus each marketplace's
  source — the live files carry absolute paths (`projectPath`, `installPath`,
  `installLocation`) and none of them may reach the repository. ⚠️ That reduction
  is held by a test which searches the written **bytes**, not the struct.
- **A recorded plugin version is a reading, not a pin.** `claude plugin install`
  has no version flag; do not invent one, and do not drop the field either — it is
  how a machine notices it is behind.

## Curation — the reason this repository is small

What is *not* tracked was measured and rejected, and the reasons are in
`petkit.yaml` beside the items and in `README.md` § *What belongs here*. The short
form: **a second copy of a file that already has an owner is a fork nobody is
watching.** Fourteen local skills are stale copies of an installed plugin; three
hook scripts are redeployed by the tools that own them; agents and commands are
addressed relative to the repository that owns them and travel with its clone.

⚠️ Before adding an item, answer the question those three refusals answer: **who
else writes this file?** If anything does, the item does not belong here.

## Testing

Tests inject the home directory as a parameter. Nothing in the logic reads `$HOME`
itself, so a test never touches the real `~/.claude` — and a test that did would
be editing the machine it is measuring.

⚠️ **A refusal test owes its positive case.** For every "this is refused" test
there is a sibling proving the same operation succeeds once the refusal condition
is removed. A guard that has never been observed to fire is not evidence that it
works; it is a line somebody believes in.

## Gates

`make check` is the gate: `gofmt`, `go vet ./...`, `go test ./... -count=1`.

⚠️ The `gofmt` step is written to capture output and exit 1 on a non-empty
result. `gofmt -l .` **prints the offending file and exits 0**, so the obvious
spelling of that step gates nothing. That defect is easy to ship and hard to
notice: the target stays green while the check does nothing, and only a CI run
against an unformatted tree reveals it.
