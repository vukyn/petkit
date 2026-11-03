# settings

Three files. `fragment.json` is **curated by hand** and petkit only ever reads
it; `plugins.json` and `mcp.json` are each written by exactly one command —
`petkit plugins capture` and `petkit mcp capture` respectively — and by
nothing else.

## `fragment.json`

The slice of `~/.claude/settings.json` this repository owns: `model`,
`permissions.defaultMode`, `skillOverrides`. `petkit settings apply` merges
exactly these keys and leaves every other key in the live file byte for byte.

## `plugins.json`

The marketplaces and plugins a machine should have. `petkit plugins plan` reads it
and prints the `claude plugin …` commands; it never installs.

**Rebuild it with `petkit plugins capture`** — which reads the live
`~/.claude/plugins/{known_marketplaces.json,installed_plugins.json}` and the
`enabledPlugins` map of `~/.claude/settings.json` (`PLUG-005`), writes
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

## `mcp.json`

The **user-scope** MCP servers a machine should have configured. `petkit mcp
plan` reads it and prints the `claude mcp add -s user …` commands; it never
configures anything itself.

⚠️ **Only user-scope is tracked.** `claude mcp add` also has `-s project`
(written into the project's own `.mcp.json`) and `-s local` (tied to one
machine AND one project path at once). Project-scope belongs to the repository
that owns it, the same reasoning `MFST-003` already applies to another
repository's agents and commands — a copy here would be a second version with
no reader. Local-scope cannot be represented by a single cross-machine record
at all: there is no "which machine, which checkout" field to put the other
half of it in. See `TODO.md` `MCP-003` for the refusal in full; re-raise only
from there.

**Rebuild it with `petkit mcp capture`** — which reads the live
`~/.claude.json`'s `mcpServers` key, writes name, transport, command, args and
URL per server plus the **NAMES** (never the values) of its env vars and
headers, prints what moved, and leaves a timestamped backup. A run that finds
nothing changed writes nothing at all.

⚠️ **Never rebuild it by copying `~/.claude.json`'s `mcpServers` map.** A live
entry's `env` and `headers` hold whatever secret was typed in when the server
was added — an API key, a bearer token — and copying either map into the
repository publishes it to everybody the repository is shared with. The
capture keeps only the key **names**, and a test searches the written bytes
for a planted fake secret value (`MCP-002`), the same discipline `PLUG-004`
holds `plugins capture` to.

⚠️ **`command` IS written as the absolute path a live machine recorded, unlike
a plugin's stripped `installPath`.** It is necessary payload: `claude mcp add`
cannot start a stdio server without knowing what to run. `mcp plan` and
`mcp capture` both print an advisory note next to it instead, saying the path
is specific to the machine that captured it.

⚠️ **A secret typed directly into `args`, rather than passed through `-e` or
`-H`, is invisible to this reduction.** `args` is positional payload — the
command line a stdio server is started with — and the reduction only knows how
to recognise and drop *names* carried in the `env` and `headers` maps. A
server added as `claude mcp add name -- some-server --token <value>` records
the value in the committed file as plainly as the command itself. This is a
known limitation, not an oversight: do not type a secret into an MCP server's
arguments if the server is meant to be captured here.

⚠️ **There is no version concept, and none was invented.** Measured
2026-11-03: `claude mcp add --help` offers `--callback-port`, `--client-id`,
`--client-secret`, `-e/--env`, `-H/--header`, `-s/--scope` and `-t/--transport`
— no version-pinning flag at all, and `claude mcp` has no analogue of
`PLUG-002`'s "last seen" reading either, because there is nothing on a live
entry that identifies a version to read. `Server` therefore has no `Version`
field, present-and-empty or otherwise.
