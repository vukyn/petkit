package version_test

import (
	"strings"
	"testing"

	"github.com/vukyn/petkit/internal/version"
)

// The version is the git tag and nothing else: a tag is printed exactly as it
// was written, and a pseudo-version is reduced to the one part of it anybody
// can act on — the revision.
func TestATagIsLeftWholeAndAPseudoVersionBecomesItsRevision(t *testing.T) {
	cases := map[string]string{
		"v1.4.0":                               "v1.4.0",
		"v1.4.0-rc.2":                          "v1.4.0-rc.2",
		"v0.0.0-20260914150405-abc123def456":   "abc123def456",
		"v1.2.3-0.20260914150405-0011aabbccdd": "0011aabbccdd",
		"":                                     version.Devel,
		"(devel)":                              version.Devel,
	}
	for raw, want := range cases {
		if got := version.Format(raw); got != want {
			t.Errorf("Format(%q) = %q, want %q", raw, got, want)
		}
	}
}

// The real reader never returns an empty string, whatever it is built with.
func TestCurrentAlwaysSaysSomething(t *testing.T) {
	if version.Current() == "" {
		t.Error("the version is empty")
	}
}

// A build from a dirty working tree carries `+dirty` after the revision. The
// pattern is anchored, so before this was allowed for, the suffix stopped the
// match and Format returned the whole pseudo-version where a revision was meant.
func TestADirtyPseudoVersionIsStillTrimmedToItsRevision(t *testing.T) {
	for _, raw := range []string{
		"v0.1.1-0.20260914063411-8d81ea5ae0fd+dirty",
		"v0.0.0-20260914063411-8d81ea5ae0fd+dirty",
	} {
		got := version.Format(raw)
		if !strings.HasPrefix(got, "8d81ea5ae0fd") {
			t.Errorf("Format(%q) = %q, want it trimmed to the revision", raw, got)
		}
		if strings.Contains(got, "20260914063411") {
			t.Errorf("Format(%q) = %q, still carries the timestamp", raw, got)
		}
	}
}

// Its positive case: a real tag is never trimmed, with or without metadata.
func TestATagWithBuildMetadataIsLeftWhole(t *testing.T) {
	for _, raw := range []string{"v0.1.0", "v0.1.0+dirty"} {
		if got := version.Format(raw); got != raw {
			t.Errorf("Format(%q) = %q, want it left whole", raw, got)
		}
	}
}
