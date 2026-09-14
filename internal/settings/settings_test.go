package settings_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/petkit/internal/settings"
)

var applyMoment = time.Date(2026, 9, 14, 15, 4, 5, 0, time.UTC)

const liveSettings = `{
  "model": "sonnet",
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh",
    "padding": 0,
    "budget": 1.50
  },
  "permissions": {
    "defaultMode": "plan",
    "allow": ["Bash(git status:*)", "Read(//Users/someone/**)"]
  },
  "feedbackSurveyState": {
    "lastShownTime": 1757000000000
  }
}
`

const fragment = `{
  "model": "opus[1m]",
  "permissions": {
    "defaultMode": "acceptEdits"
  },
  "skillOverrides": {
    "using-superpowers": "off"
  }
}
`

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func decode(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatalf("the result is not valid JSON: %v\n%s", err, content)
	}
	return value
}

// The whole point of "declared keys only": a key the fragment never mentions
// keeps its value, nested objects and all.
func TestApplyPreservesAKeyTheFragmentDoesNotMention(t *testing.T) {
	path := writeTemp(t, "settings.json", liveSettings)

	if _, err := settings.Apply(path, []byte(fragment), applyMoment); err != nil {
		t.Fatalf("apply: %v", err)
	}

	after := decode(t, readFile(t, path))
	before := decode(t, []byte(liveSettings))

	untouched := []string{"statusLine", "feedbackSurveyState"}
	for _, key := range untouched {
		if !reflect.DeepEqual(after[key], before[key]) {
			t.Errorf("%s changed:\nbefore %#v\nafter  %#v", key, before[key], after[key])
		}
	}

	// Byte-for-byte, not merely equal after a round trip: a number the fragment
	// never named keeps the literal the user wrote.
	if !bytes.Contains(readFile(t, path), []byte("1.50")) {
		t.Errorf("the literal 1.50 was rewritten:\n%s", readFile(t, path))
	}

	// And a sibling of a key the fragment DID name survives too — merging is
	// per key at every depth, not per top-level key.
	permissions, ok := after["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("permissions is %T, want an object", after["permissions"])
	}
	if !reflect.DeepEqual(permissions["allow"], before["permissions"].(map[string]any)["allow"]) {
		t.Errorf("permissions.allow changed: %#v", permissions["allow"])
	}

	// The failing case for the same operation: the declared keys DID change.
	if permissions["defaultMode"] != "acceptEdits" {
		t.Errorf("permissions.defaultMode = %v, want acceptEdits", permissions["defaultMode"])
	}
	if after["model"] != "opus[1m]" {
		t.Errorf("model = %v, want opus[1m]", after["model"])
	}
	if _, present := after["skillOverrides"]; !present {
		t.Error("skillOverrides was not added")
	}
}

// A backup is the promise that a mistake is undoable, so it has to equal the
// file that was there — byte for byte, not value for value.
func TestApplyWritesABackupEqualToTheOriginal(t *testing.T) {
	path := writeTemp(t, "settings.json", liveSettings)

	result, err := settings.Apply(path, []byte(fragment), applyMoment)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.BackupPath == "" {
		t.Fatal("no backup was written")
	}
	if !strings.HasPrefix(result.BackupPath, path+".petkit-backup-") {
		t.Errorf("the backup %q is not beside the file %q", result.BackupPath, path)
	}
	if !strings.Contains(result.BackupPath, "20260914T150405") {
		t.Errorf("the backup %q is not timestamped", result.BackupPath)
	}

	backup := readFile(t, result.BackupPath)
	if !bytes.Equal(backup, []byte(liveSettings)) {
		t.Errorf("the backup differs from the original:\nbackup:\n%s\noriginal:\n%s", backup, liveSettings)
	}
	if bytes.Equal(readFile(t, path), backup) {
		t.Error("the settings file was not changed, so the backup proves nothing")
	}
}

// Nothing is written when the file already says what the fragment says, so
// running apply twice does not accumulate backups.
func TestApplyIsIdempotentAndDoesNotBackUpAnUnchangedFile(t *testing.T) {
	path := writeTemp(t, "settings.json", liveSettings)

	if _, err := settings.Apply(path, []byte(fragment), applyMoment); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	afterFirst := readFile(t, path)

	second, err := settings.Apply(path, []byte(fragment), applyMoment.Add(time.Hour))
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if second.Changed {
		t.Error("the second apply reported a change")
	}
	if second.BackupPath != "" {
		t.Errorf("the second apply wrote the backup %s", second.BackupPath)
	}
	if !bytes.Equal(afterFirst, readFile(t, path)) {
		t.Error("the second apply rewrote the file")
	}
}

// Diff says what apply would do, and writes nothing while saying it.
func TestDiffReportsOnlyDeclaredKeysAndWritesNothing(t *testing.T) {
	path := writeTemp(t, "settings.json", liveSettings)
	before := readFile(t, path)

	changes, err := settings.Diff([]byte(liveSettings), []byte(fragment))
	if err != nil {
		t.Fatalf("diff: %v", err)
	}

	got := map[string]string{}
	for _, change := range changes {
		got[change.Path] = change.New
	}
	declared := map[string]string{
		"model":                   `"opus[1m]"`,
		"permissions.defaultMode": `"acceptEdits"`,
		"skillOverrides":          `{"using-superpowers":"off"}`,
	}
	for key, wantValue := range declared {
		gotValue, present := got[key]
		if !present {
			t.Errorf("diff does not report %s", key)
			continue
		}
		if gotValue != wantValue {
			t.Errorf("diff reports %s = %s, want %s", key, gotValue, wantValue)
		}
	}
	for _, key := range []string{"permissions.allow", "statusLine", "statusLine.padding", "feedbackSurveyState"} {
		if _, present := got[key]; present {
			t.Errorf("diff reports %s, which the fragment does not declare", key)
		}
	}
	if len(changes) != len(declared) {
		t.Errorf("diff reports %d changes, want %d: %v", len(changes), len(declared), changes)
	}

	if !bytes.Equal(before, readFile(t, path)) {
		t.Error("diff changed the file")
	}

	// The failing case: once applied, the same diff is empty.
	merged, err := settings.Merge([]byte(liveSettings), []byte(fragment))
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	again, err := settings.Diff(merged, []byte(fragment))
	if err != nil {
		t.Fatalf("second diff: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("diff after merging still reports %v", again)
	}
}

// A machine with no settings file yet gets one, and gets no backup of a file
// that never existed.
func TestApplyCreatesAMissingFileWithoutABackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")

	result, err := settings.Apply(path, []byte(fragment), applyMoment)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.BackupPath != "" {
		t.Errorf("backed up a file that did not exist: %s", result.BackupPath)
	}
	if !result.Changed {
		t.Error("apply reported no change while creating the file")
	}
	if decode(t, readFile(t, path))["model"] != "opus[1m]" {
		t.Error("the created file does not carry the fragment")
	}
}

// A live file that is not JSON is refused before anything is written: an
// unreadable file is exactly the file a backup cannot save.
func TestApplyRefusesAFileThatIsNotJSON(t *testing.T) {
	path := writeTemp(t, "settings.json", "this is not json\n")

	if _, err := settings.Apply(path, []byte(fragment), applyMoment); err == nil {
		t.Fatal("a file that is not JSON was accepted")
	}
	if got := string(readFile(t, path)); got != "this is not json\n" {
		t.Errorf("the file was changed to %q", got)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read the directory: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("the refused apply left %d files behind, want only the original", len(entries))
	}
}

// The repository's own fragment has to merge into an empty machine.
func TestTheRepositoryFragmentMerges(t *testing.T) {
	fragmentPath := filepath.Join("..", "..", "settings", "fragment.json")
	content, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Skipf("no fragment to check: %v", err)
	}
	merged, err := settings.Merge([]byte("{}"), content)
	if err != nil {
		t.Fatalf("the repository's fragment does not merge: %v", err)
	}
	if !reflect.DeepEqual(decode(t, merged), decode(t, content)) {
		t.Error("merging the fragment into an empty file did not reproduce the fragment")
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}
