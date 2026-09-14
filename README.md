# petkit

One machine's Claude Code setup, kept in a repository and installed by **symlink**.

```sh
go install github.com/vukyn/petkit/cmd/petkit@latest   # or @v0.1.0
petkit setup                                           # clone into ~/.petkit and record it
petkit sync                                            # install the links
```

`petkit setup [path]` clones the repository **this binary was built from** —
the module path is read out of the build info, so a fork installed from its own
path clones itself — records where it went, and prints what `sync` would do.
`petkit setup --sync` goes all the way in one command.

⚠️ **Setup makes no symlinks of its own.** Installing into `~/.claude` is
`sync`'s decision, with `sync`'s refusals; a command whose job is "get me
started" is the worst place to make it by surprise. Setup removes nothing
either: the path it is given must be missing or an empty directory, and anything
else is named and left alone.

**Already keep your clone somewhere else?** Then the three-step form is still
yours, and `setup` is not involved:

```sh
git clone <this repository> ~/src/petkit   # anywhere; the path is yours to pick
cd ~/src/petkit
go install ./cmd/petkit                    # the clone is the source; see below
petkit init . && petkit sync
```

Measured 2026-09-14 against a clean module cache with no `GOPRIVATE` and no
credentials: the proxy serves it (`proxy.golang.org/.../@latest` answers
`v0.1.0`), and the installed binary reports `petkit v0.1.0` — the version comes
from the tag on the proxy path exactly as it does on a local build.

⚠️ **The binary alone is still not an install** — it is one command away from
being one. Every symlink points into the clone, so the clone has to exist and
petkit has to know where it is. A binary that has neither answers `version` and
nothing else:

```
petkit: cannot find the petkit repository: no petkit.yaml above /tmp,
PETKIT_HOME is not set, and nothing is recorded in ~/.config/petkit/config.json —
run `petkit init /path/to/petkit` once, or run petkit from inside the repository
```

`petkit setup` is the answer to that message on a machine that has no clone at
all; `petkit init <path>` is the answer on a machine that does.

`petkit sync` makes `~/.claude` match `petkit.yaml`. Nothing is copied: each
target becomes a symlink into this repository, so **editing a skill here is
editing the installed one**, and `git status` sees every change you make while
working. Updating a second machine is `git pull` — there is no second step,
because there is no second copy.

| command | what it does |
|---|---|
| `petkit setup [path] [--sync]` | clone the repository this binary came from, record it, and say what `sync` would do |
| `petkit status` | one line per item: `linked`, `missing`, `stale`, `conflict` |
| `petkit sync [--dry-run]` | create the missing links, repoint the stale ones |
| `petkit doctor` | broken links, unmanaged entries, sources that do not exist |
| `petkit settings diff\|apply` | merge `settings/fragment.json` into `~/.claude/settings.json` |
| `petkit plugins plan` | print the `claude plugin …` commands this machine is missing |
| `petkit check` | the tag you are on, the newest tag upstream, whether the tree is dirty |
| `petkit version` | the build version and the item count |

## What it will not do

⚠️ **`setup` never overwrites and never links.** The directory it clones into
must be missing or empty — a directory with anything in it is named and left
exactly as it was — and without `--sync` it leaves `~/.claude` untouched.

⚠️ **`sync` only ever removes a symlink.** A real file or directory where a
target should go is a `conflict`: it is named, it is left alone, and the command
exits non-zero. Adopting an existing directory is a decision, and a tool that
makes it silently will one day make it on the wrong directory.

⚠️ **`settings apply` writes only the keys the fragment declares**, after a
timestamped backup, through a temp file and a rename. Every other key in your
live `settings.json` keeps its value byte for byte. The fragment is a slice of
preferences — model, permission mode, skill overrides — not a replacement file.

⚠️ **`plugins plan` prints; it never runs.** `claude plugin` owns plugin
installation and does it better; what this repository adds is the *list*, so a
new machine knows what it is missing.

## What belongs here, and what does not

**Here:** something you wrote, that is not owned by another tool, that you want on
every machine. Today that is two skills.

**Not here**, with the reason each was measured and rejected on 2026-09-13:

- **Fourteen skill directories** were older copies of skills an installed plugin
  already provides — and that plugin updates itself. One local copy was missing a
  whole section the plugin's copy carried; another was 64K against the plugin's
  92K. Six of the fourteen were already `off` in the settings fragment.
- **Three hook scripts** were written by the tools that deploy them — one stamps
  its own version number into the file it writes. A copy here would be overwritten
  by its owner without a word.
- **Agents, commands and repository scripts** are addressed relative to the
  repository that owns them (`$CLAUDE_PROJECT_DIR`, a Makefile variable) and
  arrive on every machine with `git clone`.

The rule behind all three: **a second copy of a file that already has an owner is
not a backup, it is a fork that nobody is watching.** `petkit.yaml` records these
refusals beside the items, so the next machine does not re-decide them.

## Version

The version is **the git tag**, read from `debug.BuildInfo` — there is no version
constant and no `-ldflags -X`. Releasing is `git tag -a` and a push; `petkit
check` compares what a machine is running against what the remote has.
