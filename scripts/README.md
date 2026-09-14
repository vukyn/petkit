# scripts

Empty on purpose.

`pet-platform/scripts/hosts.sh` and `security-scan-check.sh` were considered and
left where they are: the Makefile calls `$(SCRIPTS)/hosts.sh` and the SessionStart
hook calls `"$CLAUDE_PROJECT_DIR/scripts/security-scan-check.sh"`, so both are
addressed **relative to the repository that owns them** and travel with it on a
`git clone`. Copying them here would be a second copy of a file whose owner is
already tracked, which is the duplication this repository exists to avoid.

A script belongs here when it is **machine-level** — something a shell or a hook
runs outside any one repository. Add it as an item in `petkit.yaml` with a
`target` under `~/bin` or `~/.claude/`, never as a loose file: a script nothing
links to is a script nothing runs.
