package state_test

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/state"
)

// fakeGit answers the commands Check runs, so the answer can be tested without
// a repository, a remote or a network.
func fakeGit(answers map[string]string, failing map[string]string) state.Runner {
	return func(dir string, args ...string) (string, error) {
		key := strings.Join(args, " ")
		if message, fails := failing[key]; fails {
			return "", fmt.Errorf("%s", message)
		}
		answer, known := answers[key]
		if !known {
			return "", fmt.Errorf("unexpected command: git %s", key)
		}
		return answer, nil
	}
}

func baseAnswers() map[string]string {
	return map[string]string{
		"rev-parse --git-dir":    ".git",
		"remote":                 "origin",
		"fetch --tags --quiet":   "",
		"rev-parse --short HEAD": "abc1234",
		"describe --tags --long": "v0.2.0-0-gabc1234",
		"tag --sort=-v:refname":  "v0.2.0\nv0.1.0",
		"status --porcelain":     "",
	}
}

// A clean checkout sitting on the newest tag is the "nothing to do" answer.
func TestCheckReportsACheckoutOnTheNewestTag(t *testing.T) {
	result, err := state.Check("/repo", fakeGit(baseAnswers(), nil))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Tag != "v0.2.0" || result.Commits != 0 {
		t.Errorf("tag = %q, commits = %d, want v0.2.0 and 0", result.Tag, result.Commits)
	}
	if result.NewestTag != "v0.2.0" {
		t.Errorf("newest tag = %q, want v0.2.0", result.NewestTag)
	}
	if result.Dirty {
		t.Error("a clean tree was reported dirty")
	}
	if !result.Fetched {
		t.Error("the fetch was not reported")
	}
	if !result.UpToDate() {
		t.Error("a checkout on the newest tag is not reported as up to date")
	}
}

// Past a tag, with a dirty tree, and a newer tag upstream: the three facts the
// command exists to report.
func TestCheckReportsCommitsPastTheTagANewerTagAndADirtyTree(t *testing.T) {
	answers := baseAnswers()
	answers["describe --tags --long"] = "v0.1.0-3-gabc1234"
	answers["tag --sort=-v:refname"] = "v0.3.0\nv0.2.0\nv0.1.0"
	answers["status --porcelain"] = " M cmd/petkit/main.go\n?? notes.txt"

	result, err := state.Check("/repo", fakeGit(answers, nil))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Tag != "v0.1.0" || result.Commits != 3 {
		t.Errorf("tag = %q, commits = %d, want v0.1.0 and 3", result.Tag, result.Commits)
	}
	if result.NewestTag != "v0.3.0" {
		t.Errorf("newest tag = %q, want v0.3.0", result.NewestTag)
	}
	if !result.Dirty || result.DirtyCount != 2 {
		t.Errorf("dirty = %v with %d paths, want true and 2", result.Dirty, result.DirtyCount)
	}
	if result.UpToDate() {
		t.Error("a checkout three commits past an older tag is reported as up to date")
	}
}

// A tag containing dashes is still read correctly: the split is from the right.
func TestCheckReadsATagThatContainsDashes(t *testing.T) {
	answers := baseAnswers()
	answers["describe --tags --long"] = "v1.0.0-rc-1-2-gdeadbee"

	result, err := state.Check("/repo", fakeGit(answers, nil))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Tag != "v1.0.0-rc-1" || result.Commits != 2 {
		t.Errorf("tag = %q, commits = %d, want v1.0.0-rc-1 and 2", result.Tag, result.Commits)
	}
}

// Offline is not a failure: the fetch is reported as not having happened and
// everything already known is still said.
func TestCheckSurvivesAFailedFetch(t *testing.T) {
	result, err := state.Check("/repo", fakeGit(baseAnswers(), map[string]string{
		"fetch --tags --quiet": "could not resolve host github.com",
	}))
	if err != nil {
		t.Fatalf("an offline check returned an error: %v", err)
	}
	if result.Fetched {
		t.Error("a failed fetch was reported as done")
	}
	if !strings.Contains(result.FetchNote, "could not resolve host") {
		t.Errorf("the note does not say why: %q", result.FetchNote)
	}
	if result.NewestTag != "v0.2.0" {
		t.Errorf("newest tag = %q; the local answer should still be given", result.NewestTag)
	}
}

// A repository with no tags yet — this one, until it is first tagged.
func TestCheckHandlesARepositoryWithNoTags(t *testing.T) {
	answers := baseAnswers()
	answers["tag --sort=-v:refname"] = ""

	result, err := state.Check("/repo", fakeGit(answers, map[string]string{
		"describe --tags --long": "fatal: No names found, cannot describe anything.",
	}))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.HasTags {
		t.Error("tags were reported where there are none")
	}
	if result.Tag != "" {
		t.Errorf("tag = %q, want empty", result.Tag)
	}
	if result.UpToDate() {
		t.Error("an untagged repository cannot be up to date with a tag")
	}
}

// Somewhere that is not a git checkout gets an error, not a wrong answer.
func TestCheckRefusesSomethingThatIsNotAGitCheckout(t *testing.T) {
	_, err := state.Check("/tmp/not-a-repo", fakeGit(nil, map[string]string{
		"rev-parse --git-dir": "fatal: not a git repository",
	}))
	if err == nil {
		t.Fatal("a directory that is not a checkout was accepted")
	}
	if !strings.Contains(err.Error(), "/tmp/not-a-repo") {
		t.Errorf("the error does not name the path: %v", err)
	}
}

// git is the one external program petkit runs, and a machine without it is the
// likeliest first failure on a fresh Windows install — git is not part of that
// system the way it is in a developer's Unix shell.
//
// ⚠️ Built rather than provoked, for the same reason the Windows symlink
// failure is: this machine has git, so the only way to measure the message it
// would print without it is to hand the function the error exec returns.
func TestGitMissingFromPathIsNamed(t *testing.T) {
	missing := &exec.Error{Name: "git", Err: exec.ErrNotFound}

	err := state.DescribeGitFailure("/somewhere", []string{"clone", "https://example.com/x", "/tmp/x"}, "", missing)
	if err == nil {
		t.Fatal("a missing git produced no error")
	}
	for _, want := range []string{"git is not installed", "PATH", "petkit setup", "petkit check"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message does not mention %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "%PATH%") {
		t.Errorf("exec's own wording reached the user: %v", err)
	}
}

// The positive sibling: a git that ran and failed keeps the message it had,
// which is git's own words about what went wrong.
func TestAnOrdinaryGitFailureKeepsItsMessage(t *testing.T) {
	err := state.DescribeGitFailure("/repo", []string{"describe", "--tags"},
		"fatal: No names found, cannot describe anything.\n", errors.New("exit status 128"))

	want := "git describe --tags in /repo: fatal: No names found, cannot describe anything."
	if err == nil || err.Error() != want {
		t.Errorf("DescribeGitFailure = %v, want %q", err, want)
	}
}

// And a failure with nothing on stderr still says something.
func TestAGitFailureWithNoStderrFallsBackToTheExitError(t *testing.T) {
	err := state.DescribeGitFailure("/repo", []string{"status"}, "", errors.New("signal: killed"))

	want := "git status in /repo: signal: killed"
	if err == nil || err.Error() != want {
		t.Errorf("DescribeGitFailure = %v, want %q", err, want)
	}
}
