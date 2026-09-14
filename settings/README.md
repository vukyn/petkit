# settings

Two files, both **curated by hand**, both read by `petkit` and never written by it.

## `fragment.json`

The slice of `~/.claude/settings.json` this repository owns: `model`,
`permissions.defaultMode`, `skillOverrides`. `petkit settings apply` merges
exactly these keys and leaves every other key in the live file byte for byte.

## `plugins.json`

The marketplaces and plugins a machine should have. `petkit plugins plan` reads it
and prints the `claude plugin …` commands; it never installs.

⚠️ **Do not rebuild this file by copying `~/.claude/plugins/installed_plugins.json`.**
That file carries a `projectPath` on every project-scoped entry — an absolute path
into whatever repository the plugin was installed for, which on this machine
includes an employer's project. What belongs here is the id, the scope and the
version, and nothing else. The current file was reduced by hand for that reason,
and `PLUG-004` is the entry for giving that reduction a command so it is not
re-derived from memory.

⚠️ A `scope: project` entry is **incomplete on purpose**: without its path, a
machine cannot reinstall it into the right repository. Treat it as a record that
the plugin exists project-scoped somewhere, not as an instruction.
