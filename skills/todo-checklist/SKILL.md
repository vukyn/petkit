---
name: todo-checklist
description: Use when user requests a to-do list, checklist, smoke test sheet, QA sign-off, release checklist, or any list of items intended for human review/execution. Default output is a `.todo` file (Todo+ format) — never print the checklist to console. Guide and templates live in the skill's `todo/` folder.
---

# todo-checklist

## When to invoke

Trigger on any task whose deliverable is a list a human will tick through:

- "smoke test checklist for X"
- "release checklist", "deploy checklist", "rollback checklist"
- "QA sign-off list", "regression checklist", "review checklist"
- "todo list for ...", "task list", "things to verify"
- generic "give me a checklist of ..."

Do NOT trigger for:
- internal task tracking during *your own* execution → use TaskCreate.
- planning docs / specs → use writing-plans skill.
- commit/PR descriptions.

## Output rule (hard)

**Write a file. Do not print the checklist body to the console.**

After writing, reply with only:
1. file path
2. one-line summary (item count, scope)
3. how to open / edit (Todo+ VSCode extension or any plain text editor)

User reviewing/executing happens in the file, not chat.

## File location and name

Default: `<repo-root>/todo/<slug>.todo`

- `<slug>` = kebab-case from request (e.g. `smoke-test-login`, `release-v2.3`, `qa-soldier-fsm`).
- Create `todo/` folder if absent (`mkdir -p todo`).
- If `todo/` is gitignored or absent and project has `docs/`, fall back to `docs/todo/<slug>.todo`.
- If user names a path explicitly, honor it.

## Format — Todo+ (`.todo`)

Plain text. Symbols:

| Symbol | Meaning |
|--------|---------|
| `☐` | open / pending |
| `✔` | done |
| `✘` | cancelled / N/A |

Indent = 2 spaces. Project headings end with `:`. Tags use `@` (e.g. `@critical`, `@today`, `@1h`, `@blocked`). Nest projects to group.

## Mandatory template header — 6 lines

Every generated `.todo` file MUST start with these 6 lines verbatim (Todo+ keyboard shortcut hints). They teach the user how to operate the file in the Todo+ VSCode extension.

```
`Cmd/Ctrl+Enter` ----> `Todo: Toggle Box`
`Alt+Enter` ----> `Todo: Toggle Box`
`Alt+D` ----> `Todo: Toggle Done`
`Alt+C` ----> `Todo: Toggle Cancelled`
`Alt+S` ----> `Todo: Toggle Start`
`Cmd/Ctrl+Shift+A` ---->  `Todo: Archive`
```

Followed by a blank line, then the checklist body.

Source of truth: `todo/template.todo` in this skill folder. Copy verbatim — do not paraphrase, translate, or condense the 6 lines.

## Authoring the checklist body

When the user does **not** specify the checklist design, default to:

```
<Title>:
  Scope: <one-line scope>
  Owner: <if known, else leave blank>

  Pre-checks:
    ☐ <prerequisite 1>
    ☐ <prerequisite 2>

  <Group A>:
    ☐ <step>
    ☐ <step>

  <Group B>:
    ☐ <step>

  Sign-off:
    ☐ All items above checked
    ☐ Issues filed for any ✘
```

Heuristics for grouping when none given:
- smoke test → Pre-checks / Happy path / Edge cases / Regressions / Sign-off
- release → Pre-deploy / Deploy / Post-deploy verify / Rollback ready / Sign-off
- QA feature → Setup / Functional / Boundary / Negative / Cleanup
- generic todo → just one `Todos:` group

Each item: imperative, one verifiable outcome. No vague items ("test stuff"). Add `@critical` for blockers, `@today` for urgent, `@<duration>` (e.g. `@30m`) for estimates when meaningful.

## Resources in this skill

- `todo/README.md` — full Todo+ syntax cheat sheet + authoring guide.
- `todo/template.todo` — canonical template; the 6-line header lives here.
- `todo/examples/smoke-test.todo` — smoke test pattern.
- `todo/examples/release-checklist.todo` — release/deploy pattern.
- `todo/examples/qa-feature.todo` — QA feature sign-off pattern.

Read the relevant example before authoring; do not invent structure when one of the patterns fits.

## Workflow

1. Resolve target path (rules above).
2. Read `todo/template.todo` — copy 6-line header verbatim.
3. Pick example pattern matching the request; read it.
4. Compose body: scope line, grouped items, sign-off.
5. Write file with the Write tool. Single write, no incremental prints.
6. Reply: path + 1-line summary + open hint. Done.

## Forbidden

- Printing the checklist body in the response.
- Omitting / shortening / translating the 6-line header.
- Markdown `- [ ]` syntax (use Todo+ `☐` instead).
- Mixing GitHub-style task list and Todo+ in the same file.
- Creating the file under `internal/` or source folders.
