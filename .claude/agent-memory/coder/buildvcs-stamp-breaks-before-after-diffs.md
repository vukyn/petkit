---
name: buildvcs-stamp-breaks-before-after-diffs
description: Before/after binary output diffs must build both sides with -buildvcs=false, or Go's VCS stamp makes identical code print different versions
metadata:
  type: feedback
---

When proving a refactor changed no output, build **both** the pre-change and the
post-change binary with `go build -buildvcs=false`, and run them back to back
from the same directory.

**Why:** Go stamps `debug.BuildInfo.Main.Version` and a `+dirty` suffix from the
VCS state at build time. A binary built before the edits (clean tree) and one
built after (dirty tree) therefore disagree about any command that prints the
version or the dirty-path count — with byte-identical logic. On petkit this
produced two phantom diffs: `petkit version` printed `8d81ea5ae0fd` before and
`v0.1.1-0.20260914063411-8d81ea5ae0fd+dirty` after (the `+dirty` suffix stops
`internal/version`'s pseudo-version regex from matching), and `petkit check`
reported a different dirty-path count purely because `go get` had touched
`go.mod`. Both looked like regressions and neither was one.

**How to apply:** Any time the acceptance criterion is "the output diff must be
empty" on a Go repo here. Get the pre-change source with
`git archive HEAD | tar -x -C <scratch>` rather than stashing — stashing moves
the tree the binary is being measured against. Commands that read live state
(dirty counts, timestamps, a real `~/.claude`) have to be run from both binaries
within the same moment, or normalised out of the comparison.

Related: [[pin-comparison-base-to-worktree-head]].
