# settings

Two files. `fragment.json` is **curated by hand** and petkit only ever reads it;
`plugins.json` is written by `petkit plugins capture` and by nothing else.

## `fragment.json`

The slice of `~/.claude/settings.json` this repository owns: `model`,
`permissions.defaultMode`, `skillOverrides`. `petkit settings apply` merges
exactly these keys and leaves every other key in the live file byte for byte.

## `plugins.json`

The marketplaces and plugins a machine should have. `petkit plugins plan` reads it
and prints the `claude plugin …` commands; it never installs.

**Rebuild it with `petkit plugins capture`** — which reads the live
`~/.claude/plugins/{known_marketplaces.json,installed_plugins.json}`, writes
exactly the id, scope and version per plugin plus each marketplace's source,
prints what moved, and leaves a timestamped backup. A run that finds nothing
changed writes nothing at all.

⚠️ **Never rebuild it by copying `~/.claude/plugins/installed_plugins.json`.**
That file carries an `installPath` on every entry and a `projectPath` on every
project-scoped one — an absolute path into whatever repository the plugin was
installed for, which on a work machine is a path nobody wants published, and a
private repository is still shared with everybody who is added to it.
`known_marketplaces.json` carries an `installLocation` the same way. The capture
drops all of them, and a test feeds it a `projectPath` and searches the written
bytes for it (`PLUG-004`).

⚠️ **`version` is the version LAST SEEN on the machine that captured it, not a
version to install.** `claude plugin install` offers `--config`, `--json`,
`-s/--scope` and `-y/--yes` and **no version flag at all** (measured
2026-09-14), so nothing can be pinned through it; at least one marketplace entry
also carries `autoUpdate: true`, which would move a plugin underneath a pin if
one existed. The field is kept because a recorded reading is how a machine
notices it is running something older — `plugins plan` says so in the line where
it shows one (`PLUG-002`).

⚠️ A `scope: project` entry is **incomplete on purpose**: without its path, a
machine cannot reinstall it into the right repository. Treat it as a record that
the plugin exists project-scoped somewhere, not as an instruction. ⚠️ And note
that this is the same path the capture refuses to write: the incompleteness is
the privacy, not an oversight.
