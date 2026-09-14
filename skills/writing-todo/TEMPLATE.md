# TODO

A short index of what is done and what is not. It is deliberately **thin**:
nothing here explains a design, because the explanations already live in
`<DESIGN_DOC>` (the constraint each piece has to respect) and `<README>` (the
detail behind each question). This file exists so that "what is left" can be
read in a minute.

⚠️ **This file goes stale and nothing else in the repository is hand-kept.**
Everything else is derived — generated, tested, or regenerated. A hand-kept list
is the one thing that can quietly become a lie.
**When you finish something, tick it here in the same commit.**

⚠️ **Refer to an entry by its CODE, never by its line number.** Every item below
carries one — `ENG-006`, `SCR-008` — and § *The codes* says what the areas mean
and why a code is never reused. A line number is a fact about how much prose sits
above an item, which is exactly the thing this file grows.

⚠️ **`## Not done` is open work only.** Finishing something moves it in the same
commit: a paragraph to § *Done*, the full record to `<DECISIONS_DOC>`.

## The codes

Every entry carries a **tracking code** — `AREA-NNN` — and the sections below are
addressed by it rather than by a line number.

**A code is permanent.** Assigned once; unchanged when the item is finished, moved
or refused; **never reused** — so a code in a commit message or a PR keeps meaning
what it meant.

**Anything new gets one, at the moment it is written down.** A feature and a bug
are numbered the same way and out of the same sequence: the area says what the
subject is and the checkbox says whether it is open. Take the next free number
**by reading the index below**, not by counting entries — a retired number is
still spent.

⚠️ **A bug fixed in the same sitting it was found still gets a code**, entered
straight in as `done` with its measurements.

⚠️ **The area is about the SUBJECT, not the file the fix lands in.** Where an item
straddles two, it takes the area of the half that is still open.

| area | short for | what it covers |
|---|---|---|
| `<AAA>` | **<a…>** | <what a reader searching this area wants to find> |
| `<BBB>` | **<b…>** | <…> |

**status** is `open` (§ *Not done*), `done` (§ *Done*, record in
`<DECISIONS_DOC>`) or `refused` (§ *Decided against*).

| code | status | what it is |
|---|---|---|
| `AAA-001` | done | <one line, saying the outcome not the premise> |
| `AAA-002` | open | <one line, rewritten whenever the outcome changes> |

## Done

The honest record is the git history; the grouping below is only so the shape is
readable. One paragraph each — the measurements that settled it live in
`<DECISIONS_DOC>` under the same code.

- `AAA-001` **<Capability>.** <What exists, in two or three sentences. Numbers
  where they are load-bearing. ⚠️ a trap a reader inherits with it.>
  → `<path>`, `<doc> § <section>`.

## Not done

- [ ] `AAA-002` ⚠️ **<One sentence saying what is wrong.>** Raised YYYY-MM-DD out
      of <provenance>.

      **What is wrong.** <Mechanism. Files, functions, values. Exact error text.>

      **The evidence.** <Numbers, method, sample size.>

      | subject | measured | verdict |
      | --- | ---: | --- |
      | <…> | **<n>** | **<yes/no>** — <why> |

      ⚠️ **<A trap — especially a related item this is NOT, and why.>**

      **Why it is a sweep and not a step.** <What else moves; what may not be
      quoted until it lands.>

      → `<path>`, `<TestNameThatHoldsIt>`.

## Decided against — do not re-raise

- `AAA-004` **<The idea, in its own words>.** <Why not — with the measurement that
  refused it, not an opinion. What would have to change for it to come back.>
  → `<doc> § <section>`.
