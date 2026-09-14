// Package settings merges a curated fragment into a settings file the user
// owns.
//
// The fragment names what the repository owns; every other key in the live file
// keeps its value byte for byte. That is why this package works on
// json.RawMessage rather than on decoded values: a decode/encode round trip
// would silently rewrite numbers, reorder keys and drop the formatting of
// values nobody asked it to touch. Only whitespace is normalised, and only at
// the very end, by json.Indent.
package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Change is one key the fragment would add or alter.
type Change struct {
	Path string // dotted key path, e.g. permissions.defaultMode
	Old  string // compact JSON of the live value; empty when the key is new
	New  string // compact JSON of the value the fragment declares
}

// Added reports whether the key is absent from the live file.
func (c Change) Added() bool { return c.Old == "" }

// Diff lists what applying the fragment would change, and nothing else.
func Diff(live, fragment []byte) ([]Change, error) {
	liveObject, err := decodeObject(live, "the settings file")
	if err != nil {
		return nil, err
	}
	fragmentObject, err := decodeObject(fragment, "the settings fragment")
	if err != nil {
		return nil, err
	}
	var changes []Change
	diffObject(liveObject, fragmentObject, "", &changes)
	return changes, nil
}

// Drift is how far the file on a machine has moved from the fragment.
//
// ⚠️ It is Diff's answer with the file read for it, and nothing more. `status`
// needs the same question `settings diff` asks, and a second comparison written
// for the one-line answer would be free to disagree with the command the line
// tells the reader to run.
type Drift struct {
	// Absent: there is no settings file at all. A machine that has never run
	// `settings apply` is not drifted, it is empty, and the two want different
	// sentences.
	Absent bool

	// Changes is what applying the fragment would change — empty when the file
	// already says what the fragment says.
	Changes []Change
}

// InStep reports whether the file exists and already says what the fragment
// says.
func (d Drift) InStep() bool { return !d.Absent && len(d.Changes) == 0 }

// Inspect reads the live settings file and compares it with the fragment. A
// missing file is not an error: it is the Absent case, and it is compared
// against an empty object so the changes still describe what apply would write.
func Inspect(livePath string, fragment []byte) (Drift, error) {
	var drift Drift

	live, err := os.ReadFile(livePath)
	if os.IsNotExist(err) {
		drift.Absent = true
		live = []byte("{}")
	} else if err != nil {
		return drift, fmt.Errorf("cannot read %s: %w", livePath, err)
	}

	drift.Changes, err = Diff(live, fragment)
	if err != nil {
		return drift, err
	}
	return drift, nil
}

func diffObject(live, fragment *object, prefix string, changes *[]Change) {
	for _, key := range fragment.keys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		fragmentValue := fragment.values[key]
		liveValue, present := live.values[key]

		if !present {
			*changes = append(*changes, Change{Path: path, New: compact(fragmentValue)})
			continue
		}
		if isObject(liveValue) && isObject(fragmentValue) {
			nestedLive, errLive := decodeObject(liveValue, path)
			nestedFragment, errFragment := decodeObject(fragmentValue, path)
			if errLive == nil && errFragment == nil {
				diffObject(nestedLive, nestedFragment, path, changes)
				continue
			}
		}
		if compact(liveValue) == compact(fragmentValue) {
			continue
		}
		*changes = append(*changes, Change{
			Path: path,
			Old:  compact(liveValue),
			New:  compact(fragmentValue),
		})
	}
}

// Merge returns the live document with the fragment's declared keys applied.
// Every other key keeps its position and its value.
func Merge(live, fragment []byte) ([]byte, error) {
	if _, err := decodeObject(live, "the settings file"); err != nil {
		return nil, err
	}
	if _, err := decodeObject(fragment, "the settings fragment"); err != nil {
		return nil, err
	}
	merged, err := mergeValue(live, fragment, "")
	if err != nil {
		return nil, err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, merged, "", "  "); err != nil {
		return nil, fmt.Errorf("cannot format the merged settings: %w", err)
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

func mergeValue(live, fragment json.RawMessage, path string) (json.RawMessage, error) {
	if !isObject(live) || !isObject(fragment) {
		// A declared scalar or array replaces what is there; this is the only
		// place a live value is discarded, and the fragment named it.
		return fragment, nil
	}
	liveObject, err := decodeObject(live, path)
	if err != nil {
		return nil, err
	}
	fragmentObject, err := decodeObject(fragment, path)
	if err != nil {
		return nil, err
	}

	keys := append([]string(nil), liveObject.keys...)
	for _, key := range fragmentObject.keys {
		if _, present := liveObject.values[key]; !present {
			keys = append(keys, key)
		}
	}

	var out bytes.Buffer
	out.WriteByte('{')
	for index, key := range keys {
		if index > 0 {
			out.WriteByte(',')
		}
		out.Write(quoteKey(key))
		out.WriteByte(':')

		liveValue, inLive := liveObject.values[key]
		fragmentValue, inFragment := fragmentObject.values[key]
		switch {
		case !inFragment:
			out.Write(liveValue) // untouched, byte for byte
		case !inLive:
			out.Write(fragmentValue)
		default:
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			merged, err := mergeValue(liveValue, fragmentValue, childPath)
			if err != nil {
				return nil, err
			}
			out.Write(merged)
		}
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

// ApplyResult describes what Apply did to the file.
type ApplyResult struct {
	Path       string
	BackupPath string // empty when there was no file to back up
	Changed    bool
}

// Apply merges the fragment into the file at path: backup beside the file
// first, then a temp file in the same directory, then a rename. If the file
// already says what the fragment says, nothing is written at all.
func Apply(path string, fragment []byte, now time.Time) (ApplyResult, error) {
	result := ApplyResult{Path: path}

	live, existed, mode, err := readLive(path)
	if err != nil {
		return result, err
	}

	merged, err := Merge(live, fragment)
	if err != nil {
		return result, err
	}
	if existed && sameDocument(live, merged) {
		return result, nil
	}
	result.Changed = true

	result.BackupPath, err = WriteWithBackup(path, merged, mode, now)
	if err != nil {
		return result, err
	}
	return result, nil
}

// WriteWithBackup is petkit's whole rule for replacing a file somebody else may
// be holding: a timestamped backup beside it, then a temp file in the same
// directory, then a rename. It answers with the backup's path, or "" when there
// was nothing there to back up.
//
// ⚠️ It is exported so that every command which replaces a whole file goes
// through this one — `settings apply` and `plugins capture`. A second copy of
// these lines is a second answer to "was it backed up", and the copy that gets
// it wrong is the one nobody watched being written.
func WriteWithBackup(path string, content []byte, mode os.FileMode, now time.Time) (string, error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("cannot create %s: %w", directory, err)
	}

	backupPath := ""
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		backupPath = fmt.Sprintf("%s.petkit-backup-%s", path, now.Format("20060102T150405"))
		if err := os.WriteFile(backupPath, existing, mode); err != nil {
			return "", fmt.Errorf("cannot write the backup %s: %w — nothing was changed", backupPath, err)
		}
	case !os.IsNotExist(err):
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}

	if err := writeAtomic(path, content, mode); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}

func readLive(path string) (content []byte, existed bool, mode os.FileMode, err error) {
	info, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		return []byte("{}"), false, 0o644, nil
	}
	if statErr != nil {
		return nil, false, 0, fmt.Errorf("cannot inspect %s: %w", path, statErr)
	}
	content, err = os.ReadFile(path)
	if err != nil {
		return nil, false, 0, fmt.Errorf("cannot read %s: %w", path, err)
	}
	return content, true, info.Mode().Perm(), nil
}

// writeAtomic writes through a temp file in the same directory so a reader
// either sees the old file or the new one, never half of either.
func writeAtomic(path string, content []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temp, err := os.CreateTemp(directory, "."+filepath.Base(path)+".petkit-*")
	if err != nil {
		return fmt.Errorf("cannot create a temporary file in %s: %w", directory, err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName) // no-op once the rename has happened

	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return fmt.Errorf("cannot write %s: %w", tempName, err)
	}
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return fmt.Errorf("cannot set the mode of %s: %w", tempName, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("cannot close %s: %w", tempName, err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("cannot move %s into place at %s: %w", tempName, path, err)
	}
	return nil
}

// WriteAtomic exposes the backup-free half of the write rule for other
// packages that own a small file of their own.
func WriteAtomic(path string, content []byte, mode os.FileMode) error {
	return writeAtomic(path, content, mode)
}

// object is a JSON object kept as raw values, in the order the file wrote them.
type object struct {
	keys   []string
	values map[string]json.RawMessage
}

func decodeObject(data []byte, what string) (*object, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("cannot parse %s as JSON: %w", what, err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("%s must be a JSON object", what)
	}

	result := &object{values: map[string]json.RawMessage{}}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s as JSON: %w", what, err)
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("cannot parse %s as JSON: a key is not a string", what)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("cannot parse the value of %q in %s: %w", key, what, err)
		}
		if _, duplicate := result.values[key]; !duplicate {
			result.keys = append(result.keys, key)
		}
		result.values[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("cannot parse %s as JSON: %w", what, err)
	}
	return result, nil
}

func isObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimLeft(raw, " \t\r\n")
	return len(trimmed) > 0 && trimmed[0] == '{'
}

func compact(raw json.RawMessage) string {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return buffer.String()
}

func sameDocument(a, b []byte) bool {
	var left, right bytes.Buffer
	if json.Compact(&left, a) != nil || json.Compact(&right, b) != nil {
		return false
	}
	return left.String() == right.String()
}

func quoteKey(key string) []byte {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(key); err != nil {
		return []byte(`""`)
	}
	return bytes.TrimRight(buffer.Bytes(), "\n")
}

// SortedPaths is a printing helper: changes in a stable order.
func SortedPaths(changes []Change) []Change {
	out := append([]Change(nil), changes...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
