package state

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Runner runs a git command in a directory and returns its standard output.
// It is a parameter so the answer can be tested without a repository.
type Runner func(dir string, args ...string) (string, error)

// GitRunner is the real thing.
func GitRunner(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", DescribeGitFailure(dir, args, stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// DescribeGitFailure turns a failed git invocation into the message petkit
// prints. It is a function rather than four lines inside GitRunner so that the
// one case a machine cannot demonstrate — no git at all — can be measured on a
// machine that has git.
//
// ⚠️ The "not found" case is named rather than passed through. exec answers
// `exec: "git": executable file not found in %PATH%`, which reads like an
// internal error and does not say that installing git is the fix. It is the
// likeliest first failure on a fresh Windows machine, where git is not part of
// the system the way it is in a developer's Unix shell.
func DescribeGitFailure(dir string, args []string, stderr string, err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New(
			"git is not installed, or not on this shell's PATH — petkit runs git to clone the " +
				"repository (`petkit setup`) and to compare tags (`petkit check`); install git, " +
				"or open a shell that has it, and run the command again")
	}
	message := strings.TrimSpace(stderr)
	if message == "" {
		message = err.Error()
	}
	return fmt.Errorf("git %s in %s: %s", strings.Join(args, " "), dir, message)
}

// CheckResult is the "is there a newer version" answer.
type CheckResult struct {
	Root       string
	Fetched    bool
	FetchNote  string // why the fetch did not happen, or what it said
	Tag        string // the tag the checkout is on, or the tag it is past
	Commits    int    // commits since that tag; 0 means exactly on it
	Revision   string // short revision of HEAD
	NewestTag  string // newest tag known after the fetch
	HasTags    bool
	Dirty      bool
	DirtyCount int
}

// UpToDate reports whether the checkout is on the newest tag there is.
func (r CheckResult) UpToDate() bool {
	return r.HasTags && r.Commits == 0 && r.Tag != "" && r.Tag == r.NewestTag
}

// Check fetches tags and reports where this checkout stands. A failed fetch is
// reported and does not stop the rest: being offline should not stop petkit
// from saying what it already knows.
func Check(root string, run Runner) (CheckResult, error) {
	result := CheckResult{Root: root}

	if _, err := run(root, "rev-parse", "--git-dir"); err != nil {
		return result, fmt.Errorf("%s is not a git checkout, so there is no version to compare: %w", root, err)
	}

	remotes, err := run(root, "remote")
	switch {
	case err != nil:
		result.FetchNote = err.Error()
	case strings.TrimSpace(remotes) == "":
		result.FetchNote = "no git remote is configured, so the newest tag below is only what is already local"
	default:
		if _, err := run(root, "fetch", "--tags", "--quiet"); err != nil {
			result.FetchNote = fmt.Sprintf("could not fetch tags (%v); the newest tag below may be out of date", err)
		} else {
			result.Fetched = true
		}
	}

	if revision, err := run(root, "rev-parse", "--short", "HEAD"); err == nil {
		result.Revision = revision
	}

	if described, err := run(root, "describe", "--tags", "--long"); err == nil {
		tag, commits, ok := parseDescribe(described)
		if ok {
			result.Tag = tag
			result.Commits = commits
		}
	}

	if tags, err := run(root, "tag", "--sort=-v:refname"); err == nil {
		lines := nonEmptyLines(tags)
		if len(lines) > 0 {
			result.HasTags = true
			result.NewestTag = lines[0]
		}
	}

	if status, err := run(root, "status", "--porcelain"); err == nil {
		lines := nonEmptyLines(status)
		result.DirtyCount = len(lines)
		result.Dirty = len(lines) > 0
	}

	return result, nil
}

// parseDescribe reads `git describe --tags --long` output, which is always
// <tag>-<commits>-g<revision>; the tag itself may contain dashes, so the split
// happens from the right.
func parseDescribe(described string) (tag string, commits int, ok bool) {
	described = strings.TrimSpace(described)
	lastDash := strings.LastIndex(described, "-")
	if lastDash < 0 {
		return "", 0, false
	}
	countDash := strings.LastIndex(described[:lastDash], "-")
	if countDash < 0 {
		return "", 0, false
	}
	count, err := strconv.Atoi(described[countDash+1 : lastDash])
	if err != nil {
		return "", 0, false
	}
	return described[:countDash], count, true
}

func nonEmptyLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}
