package release

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// ReleaseCommand returns the top-level release command group.
func ReleaseCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "release",
		ShortUsage: "asc release <subcommand> [flags]",
		ShortHelp:  "Run high-level App Store release workflows.",
		LongHelp: `Run high-level App Store release workflows.

release stage prepares a version for review without submitting it. It orchestrates:
  1. Ensure/create version
  2. Apply metadata and localizations
  3. Attach selected build
  4. Run readiness checks

For the canonical App Store shipping command, use:
  asc publish appstore --app "APP_ID" --ipa app.ipa --version "VERSION" --submit --confirm

release run remains available as a deprecated compatibility pipeline when you
still want one command to prepare and submit a release. It orchestrates:
  1. Ensure/create version
  2. Apply metadata and localizations
  3. Attach selected build
  4. Run readiness checks
  5. Submit for review

After submission, monitor progress with:
  asc status --app "APP_ID"

For lower-level submission lifecycle control, use:
  asc validate --app "APP_ID" --version "VERSION"
  asc submit status --version-id "VERSION_ID"
  asc submit cancel --version-id "VERSION_ID" --confirm

` + "`asc submit preflight`" + ` remains available as a deprecated compatibility
wrapper when you need the older preflight-style text/json output.

Examples:
  asc release stage --app "APP_ID" --version "2.4.0" --build "BUILD_ID" --copy-metadata-from "2.3.2" --dry-run
  asc publish appstore --app "APP_ID" --ipa app.ipa --version "2.4.0" --submit --confirm
  asc release run --app "APP_ID" --version "2.4.0" --build "BUILD_ID" --metadata-dir "./metadata/version/2.4.0" --dry-run
  asc release run --app "APP_ID" --version "2.4.0" --build "BUILD_ID" --metadata-dir "./metadata/version/2.4.0" --confirm
  asc status --app "APP_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			ReleaseRunCommand(),
			ReleaseStageCommand(),
		},
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}
}
