package builds

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

// BuildsAppEncryptionDeclarationCommand returns the builds app-encryption-declaration command group.
func BuildsAppEncryptionDeclarationCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-encryption-declaration", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "app-encryption-declaration",
		ShortUsage: "asc builds app-encryption-declaration <subcommand> [flags]",
		ShortHelp:  "Get the app encryption declaration for a build.",
		LongHelp: `Get the app encryption declaration for a build.

Examples:
  asc builds app-encryption-declaration view --build-id "BUILD_ID"
  asc builds app-encryption-declaration view --app "123456789" --latest`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			BuildsAppEncryptionDeclarationGetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// BuildsAppEncryptionDeclarationGetCommand returns the get subcommand.
func BuildsAppEncryptionDeclarationGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("app-encryption-declaration get", flag.ExitOnError)

	selectors := bindBuildSelectorFlags(fs, buildSelectorFlagOptions{includeLegacyID: true})
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "asc builds app-encryption-declaration view (--build-id BUILD_ID | --app APP --latest | --app APP --build-number BUILD_NUMBER [--version VERSION] [--platform PLATFORM])",
		ShortHelp:  "Get the encryption declaration for a build.",
		LongHelp: `Get the encryption declaration for a build.

Examples:
  asc builds app-encryption-declaration view --build-id "BUILD_ID"
  asc builds app-encryption-declaration view --app "123456789" --latest`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := selectors.applyLegacyAliases(); err != nil {
				return err
			}
			if err := selectors.validate(); err != nil {
				return err
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("builds app-encryption-declaration get: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			buildID, err := selectors.resolveBuildID(requestCtx, client)
			if err != nil {
				return fmt.Errorf("builds app-encryption-declaration get: %w", err)
			}

			resp, err := client.GetBuildAppEncryptionDeclaration(requestCtx, buildID)
			if err != nil {
				return fmt.Errorf("builds app-encryption-declaration get: failed to fetch: %w", err)
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}
