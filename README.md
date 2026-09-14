# petkit

One machine's Claude Code setup, kept in a repository and installed by **symlink**.

```sh
git clone <this repository> ~/src/petkit   # anywhere; the path is yours to pick
cd ~/src/petkit
go install ./cmd/petkit                    # the clone is the source; see below
petkit init . && petkit sync
```

`go install` works too, and is the right way to get the binary onto a machine
that already has the clone somewhere else:

```sh
export GOPRIVATE='github.com/vukyn/*'                      # once, in your shell profile
go install github.com/vukyn/petkit/cmd/petkit@latest       # or @v0.1.0
```

⚠️ **`GOPRIVATE` is not optional while the repository is private.** Without it the
module proxy is asked for a repository it cannot read, and the failure names the
wrong cause:

```
not found: github.com/vukyn/petkit@v0.1.0: invalid version: git ls-remote …
fatal: could not read Username for 'https://github.com': terminal prompts disabled
```

With `GOPRIVATE` set, Go goes to GitHub directly and uses the git credentials the
machine already has (`gh auth login` is enough). Measured 2026-09-14: both
`@latest` and `@v0.1.0` install, and the binary reports `petkit v0.1.0` — the
version comes from the tag either way.

⚠️ **The binary alone is not an install.** Every symlink points into the clone, so
the clone has to exist and `petkit init` has to know where it is. A binary with no
repository answers `version` and nothing else:

```
petkit: cannot find the petkit repository: no petkit.yaml above /tmp,
PETKIT_HOME is not set, and nothing is recorded in ~/.config/petkit/config.json —
run `petkit init /path/to/petkit` once, or run petkit from inside the repository
```

`petkit sync` makes `~/.claude` match `petkit.yaml`. Nothing is copied: each
target becomes a symlink into this repository, so **editing a skill here is
editing the installed one**, and `git status` sees every change you make while
working. Updating a second machine is `git pull` — there is no second step,
because there is no second copy.

| command | what it does |
|---|---|
| `petkit status` | one line per item: `linked`, `missing`, `stale`, `conflict` |
| `petkit sync [--dry-run]` | create the missing links, repoint the stale ones |
| `petkit doctor` | broken links, unmanaged entries, sources that do not exist |
| `petkit settings diff\|apply` | merge `settings/fragment.json` into `~/.claude/settings.json` |
| `petkit plugins plan` | print the `claude plugin …` commands this machine is missing |
| `petkit check` | the tag you are on, the newest tag upstream, whether the tree is dirty |
| `petkit version` | the build version and the item count |

## What it will not do

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
