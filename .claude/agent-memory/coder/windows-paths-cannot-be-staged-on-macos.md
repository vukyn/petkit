---
name: windows-paths-cannot-be-staged-on-macos
description: A Windows path layout resolved on a macOS filesystem produces names the host cannot hold in the right shape, so end-to-end Windows tests are impossible here — extract the pure decision and test that instead
metadata:
  type: feedback
---

When testing Windows behaviour from macOS by passing GOOS in as a parameter,
**anything that touches the real filesystem cannot be staged.** Test the pure
decision directly instead, in a `package <pkg>` internal test file if the
decision is unexported.

**Why:** `path/filepath` is compiled for the host, so a Windows layout resolved
on macOS yields hybrids like `<home>/.claude/skills\writing-todo` — one file
literally named `skills\writing-todo`, whose parent is `.claude`, not
`.claude/skills`. Any test that then asks the filesystem a question ("is this
directory entry one the manifest claims?") is incoherent: the directory the
survey reads and the path the manifest resolves to can never be parent and
child on this host. In petkit this bit twice — a symlink fixture reported
`missing` because the link had been created at a hand-spelled path instead of
the one the layout resolved, and doctor's managed-entry lookup survived its
mutation because the end-to-end test could not reach the branch at all.

**How to apply:** Two rules. (1) In a fixture, always create the file at
`layout.Resolve(...)` — the path the code under test will ask for — never at a
path spelled independently in the test; otherwise you measure the fixture.
(2) When the branch needs both a Windows path *and* a real file, it cannot be
reached here: lift the decision into a pure function over strings, test that,
and file the filesystem wiring as explicitly unmeasured with the exact command
the owner should run on the real machine. Do not claim the wiring is covered.

Assertions that must hold on both hosts should be written as
`filepath.Join(root, "a\\b")` — the host contributes its own separator between
the two halves, and the remainder still proves the conversion happened.

Related: [[buildvcs-stamp-breaks-before-after-diffs]].
