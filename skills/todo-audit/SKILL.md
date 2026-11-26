---
name: todo-audit
description: Audit a TODO.md written with the writing-todo skill — rule violations, orphaned or lost entries, index/entry drift, stale claims, and what to clean up or tighten. Read-only; reports and proposes, never edits unless asked. Use for "audit todo", "verify TODO.md", "kiểm tra todo".
---

# Auditing a TODO.md

Checks one `TODO.md` against the rules in the **writing-todo** skill
(the sibling `writing-todo` skill directory). Read that file first if its rules are not
already in context: this skill only checks them, it does not restate them.

**Read-only.** Report and propose. Edit `TODO.md` only when the user asks, and then in
the repository's normal flow (its own branch / worktree, one commit, a PR).

Answer in the user's language. Codes, file names and quoted text stay verbatim.

## 1. Mechanical pass — run the script, never eyeball it

`<this skill's base directory>` is the path shown when the skill loads; `todo.py` lives in the
sibling `writing-todo` skill.

```sh
python3 <this skill's base directory>/../writing-todo/todo.py audit [path/to/TODO.md]
```

With no path it finds `TODO.md` in the current directory or up to the repository root.
It prints JSON and exits 1 when there is any error. It checks:

| check | what it catches |
|---|---|
| sections | the four `##` sections present once each, in order |
| index | duplicate rows, a status outside `open` / `done` / `refused`, an area the area table does not define |
| index ↔ entries | an open code with no `- [ ]` entry under *Not done* (a lost title line), a done code still under *Not done*, a refused code missing from *Decided against*, an entry with no index row, a code with two entries |
| ticks | `- [x]` under *Not done* (closing must move it), an unticked entry under *Done* |
| orphans | a `Raised …` body that starts a block with no title above it, or a second `Raised` inside one entry — the body of an entry whose title line was lost |
| references | line-number references, codes mentioned in prose that have no index row |
| refusals | a refused entry that does not say what would bring it back |

It also returns `counts`, `next_free` per area, the largest entries and the file size —
use them in step 3.

⚠️ The script decides only what text can decide. A clean run means "no mechanical
defect", not "the file is right". Never report it as the whole audit.

## 2. Judgment pass — what the script cannot see

Read the entries the script flagged, then sample the open ones. For each, check:

- **Claim without evidence.** An open entry stating something is slow, broken or
  unbalanced with no number, method or sample size, and not marked unmeasured.
- **Stale against the code.** An open entry whose problem the code no longer has.
  Verify before calling it stale: grep the file / function the entry points at (`→`
  line), or run the one command that settles it. Say what you checked. If you did not
  verify, say "có thể đã lỗi thời" / "possibly stale", never "stale".
- **Index row out of date.** The one-line summary still states the premise after the
  outcome changed (the skill's rewrite-the-row rule).
- **Contradictions.** A later paragraph contradicting an earlier sentence instead of
  correcting it in place; a sentence saying an entry "stays open" in a closed entry.
- **Duplicates and overlaps.** Two open entries describing the same work, or an
  umbrella entry (e.g. "more effects") whose concrete parts already have their own codes.
- **Traps marked ⚠️** that are no longer traps, or ⚠️ used as decoration.
- **Format the skill does not use** (a fourth status such as `shipped`, `Done` bullets
  for plain bug fixes, sections renamed): name it as a deviation from the skill, not as
  a bug in the project, and say what converting it would involve.

## 3. Report

Lead with the verdict in one line: errors / warnings / judgment findings, counts.

Then, most severe first, one line per finding:

```
<code or line> — <what is wrong> — <proposed fix>
```

Group as **Lỗi cần sửa** (script errors and verified judgment defects), **Nên dọn**
(warnings, stale or duplicated entries, out-of-date rows), **Tối ưu** (size: the largest
entries and whether their detail belongs in `docs/decisions.md` or a spec; a *Done*
section grown into a changelog; the next free code per area if the user is about to add
items).

End by offering to apply the fixes in one PR. Do not apply them unasked.
