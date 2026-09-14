# Decisions

Every piece of work that is **finished**, with the reasoning and the measurements
that decided it. `TODO.md` says what is open; this says why a finished thing is
the way it is. The entries carry the same `AREA-NNN` codes.

- [x] `MFST-004` ⚠️ **`~/` is a prefix, not a fence — and the rule that caught an
      escaping target was in the wrong place.** Found and fixed 2026-09-14, while
      reviewing the first version before it was committed.

      **What was wrong.** A target had to start with `~/`, and `~/../elsewhere/x`
      does. Expanded, it is `filepath.Join(home, "../elsewhere/x")` — the parent of
      the home directory. Validation accepted it; only `doctor` objected, and
      `doctor` is a command somebody runs **after** `sync` has already created the
      link. The rule existed and was enforced one step too late.

      **What shipped.** `validateTarget` refuses a target whose cleaned remainder
      climbs out (`..` or `../…`), naming the item and quoting the target. Two
      tests: the refusal, and its positive case — `~/.claude/skills/../skills/x`
      **is** accepted, so the guard refuses the escape rather than the two dots.
      Mutation-checked: deleting the four-line guard turns
      `TestATargetThatClimbsOutOfHomeIsRefused` red.

      ⚠️ **`doctor`'s own outside-home branch was deleted, not kept as a
      backstop.** Once validation refuses the shape, no manifest that reaches
      `Doctor` can carry it, so the branch could not fire — and a guard that cannot
      fire is not defence in depth, it is a line that reads like protection. Its
      test went with it, and the reason is written where the branch was.
      `manifest.UnderHome`, the exported helper it used, had no other caller and
      was removed too.

      **The tell, for next time.** The brief said "a target must start with `~/`"
      and the implementation said exactly that. The gap was between two rules that
      each looked complete: validation owned the *shape* and doctor owned the
      *location*, and nothing owned "a shape that produces a bad location". When
      two checks split one property, ask which of them runs before the action.
