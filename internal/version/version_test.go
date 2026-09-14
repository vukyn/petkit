package version_test

import (
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
