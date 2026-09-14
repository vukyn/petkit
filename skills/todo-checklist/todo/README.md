# todo/ — guide + examples for the `todo-checklist` skill

This folder is the source of truth for how Claude generates `.todo` files. The parent `SKILL.md` references it.

## Files

- `template.todo` — canonical template. The first 6 lines are the **mandatory header** (Todo+ shortcut hints) that MUST appear verbatim on every generated `.todo` file.
- `examples/smoke-test.todo` — smoke test layout.
- `examples/release-checklist.todo` — release / deploy / rollback layout.
- `examples/qa-feature.todo` — feature QA sign-off layout.

## Todo+ syntax (the format we emit)

Plain text. UTF-8. Indent = 2 spaces.

### Symbols

| Symbol | Meaning |
|--------|---------|
| `☐` | open / pending |
| `✔` | done |
| `✘` | cancelled or N/A |

### Projects

Any line ending with `:` is a project. Nest by indent.

```
Smoke test:
  Auth:
    ☐ Login with valid creds
    ☐ Login with invalid creds
  Payments:
    ☐ Stripe checkout 200 OK
```

### Tags

`@name` or `@name(value)`. Useful ones:

- `@critical`, `@high`, `@low`, `@today` — priority
- `@blocked(<reason>)` — paused
- `@owner(<name>)` — assignee
- `@30m`, `@1h30m`, `@est(2 days)` — estimate
- `@started(YY-MM-DD HH:mm)`, `@done(YY-MM-DD HH:mm)` — timestamps (extension auto-fills via shortcut)

### Formatting

`*bold*` `_italic_` `~strike~` `` `code` `` — markdown-lite, optional.

## Authoring rules (what Claude follows)

1. **Header is non-negotiable.** Copy the 6 lines from `template.todo` verbatim. No translation, no condensation.
2. **Imperative items.** Each `☐` line = one verifiable action / outcome. Bad: `☐ test login`. Good: `☐ Log in with valid creds → land on /dashboard`.
3. **Group by phase.** Pre-checks → execution groups → sign-off. Pick a pattern from `examples/` when one fits.
4. **No console dump.** The skill always writes a file; the response is just `path + summary`.
5. **One file per checklist.** Don't append unrelated checklists to the same file.

## Default file location

`<repo-root>/todo/<slug>.todo` — create the folder if missing. If a project has its own convention, honor it.

## Opening generated files

- VSCode + [Todo+ extension](https://marketplace.visualstudio.com/items?itemName=fabiospampinato.vscode-todo-plus) → full shortcut + statistics support.
- Any plain text editor → still readable, manual symbol toggling.
