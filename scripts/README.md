# scripts

Empty on purpose.

The scripts considered first were a workspace's own — one called from its
Makefile, one from a `SessionStart` hook as
`"$CLAUDE_PROJECT_DIR/scripts/<name>.sh"`. Both are addressed **relative to the
repository that owns them** and travel with it on a `git clone`, so copying them
here would be a second copy of a file whose owner is already tracked — the
duplication this repository exists to avoid.

A script belongs here when it is **machine-level** — something a shell or a hook
runs outside any one repository. Add it as an item in `petkit.yaml` with a
`target` under `~/bin` or `~/.claude/`, never as a loose file: a script nothing
links to is a script nothing runs.
