# Memory Index

One line per memory: link + hook only. Detail lives in the topic file.

## Verification technique
- [buildvcs stamp breaks before/after diffs](buildvcs-stamp-breaks-before-after-diffs.md) — build both sides `-buildvcs=false`; `+dirty` fakes a version regression
- [Windows paths cannot be staged on macOS](windows-paths-cannot-be-staged-on-macos.md) — GOOS-as-parameter tests break the moment they touch the FS; test the pure decision
