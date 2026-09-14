---
name: petkit-gosec-baseline
description: petkit gosec baseline (2026-09-14) — 17 hits, all design-as-intended/FP; what bounds each, and the one history-only path leak in v0.1.0 README
metadata:
  type: project
---

First scan 2026-09-14: govulncheck 0, osv 0, gitleaks 0 (17 commits), gosec 17 — none real.

- G703/G304/G306 on `internal/state/locate.go` `Init`: path = `ConfigPath(home)` where `home` is
  `os.UserHomeDir()` joined to a constant — no user input reaches the path. 0644 is right: the file
  holds only `{"repo": <path>}`.
- G304 (8×): every `os.ReadFile` reads `<repo-root>/petkit.yaml|settings/*.json` or
  `~/.claude/settings.json`; root comes from `manifest.Load` which `Validate`s targets to `~/`
  (rejects `..`, absolute, backslash) and sources to inside the root.
- G301 0755 dirs: `~/.claude`, `~/.petkit` parents — must be traversable; correct.
- G204 `internal/state/check.go:18` `GitRunner`: `exec.Command("git", args...)`, no shell; clone URL
  is `"https://" + debug.BuildInfo.Main.Path` — attacker would need to build the binary, i.e. already
  runs code. Not an injection surface.
- G104 `settings.go` `writeAtomic`: `temp.Close()` on already-failing branches. FP.
- Symlink/remove design: `link.go` `repointLink` re-`Lstat`s immediately before `os.Remove` and
  refuses anything not a symlink; `Plan` only ever proposes removing a symlink. Design-as-intended.

**Why:** gosec re-reports all 17 every run; without this the triage is re-done from the code.
**How to apply:** re-verify the guard lines still exist (`grep -n 'ModeSymlink == 0' internal/link/link.go`,
`grep -n 'validateTarget' internal/manifest/manifest.go`) then mark as baseline; only a *new* rule id
or a new file is worth reading.

History-only: v0.1.0 README carried `~/vukyn/repo/pet-platform/petkit` (workspace layout, no
username, no secret); removed in `d47225b`, still in history + proxy.golang.org. Informational,
do not re-flag above Low. `/Users/someone` and `/home/me` are test fixtures.
