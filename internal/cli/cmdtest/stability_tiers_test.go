package cmdtest

import (
	"strings"
	"testing"
)

// TestExperimentalCommandsHaveStabilityLabel ensures every command surface
// that is marked experimental carries a consistent "[experimental]" prefix in
// its ShortHelp so that the label is visible in grouped root help, subcommand
// listings, and generated docs.
func TestExperimentalCommandsHaveStabilityLabel(t *testing.T) {
	root := RootCommand("1.2.3")

	cases := []struct {
		path []string // subcommand path from root
	}{
		{[]string{"web"}},
		{[]string{"screenshots", "run"}},
		{[]string{"screenshots", "capture"}},
		{[]string{"screenshots", "frame"}},
		{[]string{"screenshots", "list-frame-devices"}},
		{[]string{"screenshots", "review-generate"}},
		{[]string{"screenshots", "review-open"}},
		{[]string{"screenshots", "review-approve"}},
	}

	for _, tc := range cases {
		cmd := findSubcommand(root, tc.path...)
		if cmd == nil {
			t.Errorf("command %v not found", tc.path)
			continue
		}
		if !strings.HasPrefix(cmd.ShortHelp, "[experimental]") {
			t.Errorf("command %v: expected ShortHelp to start with [experimental], got %q", tc.path, cmd.ShortHelp)
		}
	}
}
