---
name: writing-todo
description: Use when creating or maintaining a project's TODO.md — a coded, measurement-backed index of open work, finished work and refused work. Covers bootstrapping the file for a new repo, assigning tracking codes, writing an entry, and closing one.
---

# Writing a TODO.md

## Overview

A `TODO.md` written this way is **an index of what is open, plus the evidence that
settled everything else**. It is not a task list and not a design document.

Three properties make it worth the effort:

1. **Every item has a permanent code** (`AREA-NNN`), so a commit, a PR, a code
   comment and a later note can name the same thing forever.
2. **Every claim carries its measurement.** An entry that says a thing is slow,
   unbalanced or broken states the number, the method and the sample size. An
   entry with no number is an opinion, and opinions go stale silently.
3. **A refusal is an entry, not a deletion.** Work decided against keeps its code
   and its reasoning under *Decided against — do not re-raise*, so the same idea
   is not re-proposed every six months.

**Announce at start:** "I'm using the writing-todo skill."

Write the file in the same language and register as the repo's other prose
(README, CLAUDE.md). The examples below are English; translate the section names
if the project's docs are in another language, but keep the code format
(`AREA-NNN`) verbatim — it is an identifier, not prose.

## When to use

- A repo has no `TODO.md` and wants one → **Bootstrapping** below.
- A new feature, bug, or question is raised → **Writing an entry**.
- Work finishes, or is decided against → **Closing an entry**.

Do **not** use this for a session-scoped scratch list. This file is committed and
read by strangers; ephemeral steps belong in the agent's own todo tool.

## The file's shape

Four sections, in this order:

```markdown
# TODO

<what this file is; what it deliberately does NOT explain, and where that lives>
⚠️ <the staleness warning — "tick it here in the same commit">
⚠️ <refer by CODE, never by line number>

## The codes
<the area table, the permanence rule, the status vocabulary, and THE INDEX>

## Done
<one short paragraph per finished item — the readable shape of what exists>

## Not done
<the working section: open items AND recently-closed ones with their full record>

## Decided against — do not re-raise
<refusals, each with the measurement that refused it>
```

⚠️ **The headings must mean exactly what they say, and one rule keeps them that
way: closing an entry MOVES it.** `## Not done` holds open items and nothing else.
In the same commit that finishes the work, the entry leaves that section — a short
paragraph joins `## Done`, and the full record with its measurements goes to
`docs/decisions.md`.

⚠️ **Do not leave a finished entry ticked in place "until it is distilled later".**
That arrangement reads as harmless and is not. The source this skill was drawn
from tried it and stated in its own header that *"`## Not done` is only what is
not done"*; eight days later that section held **30 finished entries against 12
open ones** — 2,663 lines of shipped work under a heading saying it was not
shipped. A staging area with no deadline is never emptied.

**A small project without a decisions doc** gets a simpler destination, not an
exception: keep the full record inside the `## Done` entry itself. The invariant
does not move — nothing finished stays under *Not done*.

## The codes

`AREA-NNN` — three or four letters for the subject area, three digits.

**Rules, all of them non-negotiable:**

- **Permanent.** A code is assigned once. It does not change when the item is
  finished, moved between sections, or refused.
- **Never reused.** A retired number stays spent.
- **Everything gets one, at the moment it is written down.** A bug found and fixed
  in the same sitting still gets a code, entered straight in as done with its
  measurements — the code exists so a commit and a later note can name it.
- **One sequence per area.** No second prefix for bugs vs features: the area says
  the subject, the status says whether it is open.
- **Take the next free number by reading the index, never by counting entries** —
  retired numbers are still spent.
- **The area is about the SUBJECT, not the file the fix lands in.** A rating
  defect fixed in the combat package is still a rating item, because what a
  reader searches for is everything about how the opponent chooses. An item that
  straddles two areas takes the area of the half that is still open.

**Choosing areas for a new project:** 5–9 of them, each a thing a reader would
search for as a subject. Derive them from the top-level packages/domains, then
collapse any that nobody would ever search separately. Write a table with a
`what it covers` column — and if any area's letters read as an unrelated English
word, say so explicitly in that column.

```markdown
| area | short for | what it covers |
|---|---|---|
| `ENG` | **eng**ine | the engine's own rules and arithmetic |
| `SCR` | **scr**eens | every screen, both clients, i18n |
| `NET` | **net**work | rooms, the socket, the wire protocol |
```

**Status vocabulary** — three words, defined in the header:

| status | meaning | lives in |
|---|---|---|
| `open` | not done | *Not done*, `- [ ]` |
| `done` | finished | *Done* (summary) + `docs/decisions.md` (record) |
| `refused` | decided against | *Decided against* |

⚠️ **Three, not four.** A separate word for "finished long ago" buys nothing once
closing moves the entry: both would name the same section, and the index would
carry two words for one fact.

**The index** is a table of every code ever issued, with its status and a
one-line summary. It is the file's table of contents and the only thing that
knows which numbers are spent.

```markdown
| code | status | what it is |
|---|---|---|
| `ENG-007` | open | Four ways of playing the board the engine cannot express yet |
| `ENG-008` | refused | Re-rolling the turn-order tie-break from the seed |
| `ENG-014` | done | Nothing runs the test suite on a pull request — DONE. ⚠️ Its first line never gated: `gofmt -l` lists and exits 0 |
```

⚠️ **Rewrite the index row when the outcome changes.** An entry that was raised
as a bug and closed as "the premise did not survive measurement" must say so in
its row — otherwise the index advertises a defect that does not exist.

## Writing an entry

An entry is a claim, its provenance, and its evidence.

```markdown
- [ ] `AREA-NNN` ⚠️ **One sentence saying what is wrong, in the indicative.**
      Raised YYYY-MM-DD out of <where it came from — an item, a question, a
      failing run>.

      **What is wrong.** The mechanism, concretely. Name files, functions and
      values. Quote the exact error if there is one.

      **The evidence.** The numbers, the method, the sample size.

      | subject | measured | verdict |
      | --- | ---: | --- |
      | ... | **85** | **no** — 14× the allowance |

      ⚠️ **A trap a reader would otherwise walk into.** Especially: a related
      item this is NOT, and why — so nobody re-opens the wrong one.

      **Why it is a sweep and not a step** (when true). What else moves if this
      is touched, and what may not be quoted until it lands.

      → `path/to/file.go`, `docs/thing.md` § Section, `TestNameThatHoldsIt`.
```

Rules for the body:

- **Bold the claim, not the noise.** The first bold sentence is what a reader
  takes away if they read nothing else.
- **⚠️ marks a trap only** — something that would cause a wrong action. Do not
  decorate ordinary statements with it, or it stops being a signal.
- **Every number carries its method.** `81.3% over 10,000 seeds, band ±0.8pp`,
  not `much better`. If the figure is a reading of a particular fixture rather
  than of the thing itself, say that in the entry — it is the difference between
  a measurement and a claim.
- **A null result is a result.** "Built, measured, moved nothing, reverted" is a
  full entry and saves the next person the same week.
- **Record the misreadings.** If the problem was diagnosed wrongly on the way,
  write down what was believed, what settled it, and the tell that should have
  been followed. That paragraph is worth more than the fix.
- **Point, do not repeat.** Link to the design doc, the test name, the source
  line. The entry says what is left; it does not re-explain the architecture.
- **Length is free; staleness is not.** A twelve-kilobyte entry carrying the
  measurements that settled a question is the file working. A one-line entry
  whose evidence lives only in someone's memory is the file failing.

## Closing an entry

All of it in **one commit, the same one as the work**:

1. Tick the checkbox and **move the entry out of `## Not done`.** The full record
   — measurements, traps, the misreadings — goes to `docs/decisions.md` under the
   same code. Never a follow-up commit: a move deferred is a move that does not
   happen.
   ⚠️ **`## Done` does not get a bullet per closed item.** It describes
   capabilities a reader can use, so a new capability earns a paragraph there and
   a bug fix or a settled measurement does not — its one-line summary is already
   the index row, and a third wording of the same fact is the duplication this
   file exists to avoid.
2. Rewrite the opening sentence into the past: what shipped, and what the
   measurement finally said. Keep the original claim visible if it was wrong —
   **correct a false sentence in place, do not contradict it in a later
   paragraph.**
3. Update the index row so its summary matches the outcome, not the premise the
   item was raised on.
4. A refused item goes to *Decided against* instead, **with the measurement that
   refused it**, and says what would have to change for it to come back.
5. Give a fresh code now to every open question that fell out of the work.

**Check it held** — this must print nothing:

```sh
awk '/^## Not done/{f=1;next} /^## /{f=0} f && /^- \[x\]/' TODO.md
```

## Anti-patterns

| ❌ | ✅ |
|---|---|
| "See the TODO at line 412" | "See `DAT-007`" |
| "Performance is bad" | "141‰ endless of 600 battles, 14× the allowance" |
| Deleting a rejected idea | An entry under *Decided against* with the numbers |
| A second prefix: `BUG-003` vs `FEAT-003` | One sequence per area; the checkbox says which |
| Reusing a retired number | Take the next free one from the index |
| Ticking a finished entry in place | Move it out of *Not done* in the same commit |
| Ticking the box in a later commit | Same commit as the work |
| ⚠️ on every paragraph | ⚠️ only where a reader would act wrongly |
| An entry that re-explains the design | A pointer to the doc that owns it |

## Bootstrapping a new project

1. Read the repo: top-level packages, the docs that already exist, the git log's
   recurring subjects.
2. Choose the areas. Write the area table.
3. Write the header: what this file is, what it deliberately does not explain and
   where that lives instead, the staleness warning, and the code-not-line-number
   rule.
4. Seed *Done* from what already exists — one short paragraph per major
   capability, each with a code. This is what makes the file readable on day one.
5. Seed *Not done* from what is genuinely open — **open only**. Give each a code
   and at least a claim; add the evidence as it is measured.
6. Seed *Decided against* from any argument the project has already had. If none
   have been had yet, leave the heading with one line saying so.
7. Cross-link: add a line to `CLAUDE.md`/`README.md` saying that `TODO.md` is the
   open list, addressed by code.

A full skeleton to copy: `TEMPLATE.md` beside this file.

## Checklist before committing a TODO change

- [ ] Every new item has a code, taken from the index rather than counted.
- [ ] The index row exists and its summary matches the entry's current outcome.
- [ ] Every claim in the entry has a number, a method and a sample size, or is
      explicitly marked as unmeasured.
- [ ] Nothing is referred to by line number.
- [ ] Closed items are ticked **and moved** in this same commit.
- [ ] `## Not done` holds zero `- [x]` entries — run the awk check above.
- [ ] Refused items say what would have to change to re-raise them.
