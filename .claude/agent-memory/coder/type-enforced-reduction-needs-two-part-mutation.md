---
name: type-enforced-reduction-needs-two-part-mutation
description: When a struct's undeclared fields are what stops data leaking, a one-line mutation cannot fire the guard — declare the field AND use it, or the test looks untested
metadata:
  type: feedback
---

When the safety property is "this field never reaches the output" and it is
enforced by **not declaring the field** on the decode struct, the mutation that
proves the test needs **two** edits: declare the field back on the type, *and*
pass it into a field that is written. Declaring it alone changes nothing, and a
mutation that changes nothing reads exactly like a guard that does not work.

**Why:** In petkit's `plugins capture` (PLUG-004) the live
`installed_plugins.json` carries a `projectPath` — an absolute path into a
private repository — and the capture is safe because `liveInstalled` has no such
field, so `encoding/json` drops it on decode. The first mutation attempt added
`ProjectPath string` to the type and the test still passed, which momentarily
looked like the leak test was worthless. It was not: an unused field is not a
leak, which is the whole property the design was after. Adding
`Scope: entries[0].Scope + entries[0].ProjectPath` made it fail immediately, with
the path quoted in the failure message.

**How to apply:** Any guard whose mechanism is absence rather than a branch —
undeclared struct fields, an allowlist, a type with nowhere to put the bad value.
Mutate the *data flow*, not the declaration. Both edits still compile, so this
stays inside the "change a value, not a symbol" rule. And assert on the **bytes
that would be written**, not on the struct: a reduction only visible by reading
the code is the hand-reduction it was meant to replace.

Related: [[buildvcs-stamp-breaks-before-after-diffs]].
