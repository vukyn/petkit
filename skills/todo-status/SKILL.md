---
name: todo-status
description: Present the open (Not done) items of a TODO.md written with the writing-todo skill, grouped by area with code and a one-line summary, flag in-progress or possibly stale items, and recommend what to do next. Read-only. Use for "todo status", "what's open", "còn gì chưa làm", "làm gì tiếp".
---

# Open TODO items, and what to do next

Lists what is still open in one `TODO.md` (written with the **writing-todo** skill) and
ends with a recommendation. **Read-only**: it never edits the file or starts the work.

Answer in the user's language. Codes, file names and identifiers stay verbatim.

## 1. Collect

`<this skill's base directory>` is the path shown when the skill loads; `todo.py` lives in the
sibling `writing-todo` skill.

```sh
python3 <this skill's base directory>/../writing-todo/todo.py status [path/to/TODO.md]
```

JSON: the open items grouped by area in the area table's order, each with `code`,
`summary` (the index row), `line` (its entry) and `mentions` (other codes its entry
names — dependencies and related work). With no path it finds `TODO.md` up to the
repository root.

If the script reports no open items, or the file has no index, say so and stop. If the
user did not run `todo-audit` recently and the listing looks inconsistent (an open code
with `line: null`), mention once that `todo-audit` would explain it.

Also gather, cheaply:

- `git worktree list` and `git branch --list` — a branch or worktree whose name carries
  a code, or obviously matches an item, marks that item **in progress**.
- For an item whose summary already sounds settled ("may be stale", a dependency now
  `done`, a fix the recent `git log --oneline -20` names), read its entry. Do not open
  every entry.

## 2. Present

One block per area. Heading: the area's name in the user's language with its code in
parentheses when helpful (`Hiệu ứng lá (EFX)`), bold. Then one line per item:

```
- <CODE>: <summary rewritten as one short sentence in the user's language>.
```

Keep the index order inside an area. Rewrite, do not translate word for word: the
reader wants what the item *is*, not the row's wording.

Append to the line, only when true and checked:

- **Đang làm** ở worktree `<name>` / branch `<name>`;
- **Có thể đã lỗi thời**, and the one reason (what you saw in the code or the log);
- **Cần <CODE> trước** when its entry says it depends on another open item;
- **Chờ chủ dự án quyết** when the entry says a decision is pending.

No other commentary per line. No tables: the block layout is the format.

## 3. Recommend

End with one short paragraph, **Mục đáng làm tiếp: <CODE>**, and why, in one or two
sentences. Prefer, in this order:

1. an item that **unblocks** other open items (its code appears in their `mentions` /
   "needs" text) — say which ones it unblocks;
2. a cheap item that closes a correctness or security gap;
3. an item the owner already decided ("owner chose …") and is only waiting to be built;
4. otherwise the smallest item that ships a visible capability.

Never recommend an item marked in progress elsewhere, or one waiting for an owner
decision, as "next" — name the decision instead if it blocks the most. Name at most one
alternative, with its reason in a clause.
