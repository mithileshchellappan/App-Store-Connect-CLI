package cmdtest

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

const (
	publishAppStoreDeprecationWarning = "Warning: `asc publish appstore` is deprecated. Use `asc release run`."
	submitCreateDeprecationWarning    = "Warning: `asc submit create` is deprecated. Use `asc release run`."
)

func TestPublishHelpShowsCanonicalTestFlightSurface(t *testing.T) {
	stdout, stderr, runErr := runRootCommand(t, []string{"publish"})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !usageListsSubcommand(stderr, "testflight") {
		t.Fatalf("expected publish help to list testflight, got %q", stderr)
	}
	if usageListsSubcommand(stderr, "appstore") {
		t.Fatalf("expected publish help to hide deprecated appstore path, got %q", stderr)
	}
	if !strings.Contains(stderr, "asc release run") {
		t.Fatalf("expected publish help to point App Store users to asc release run, got %q", stderr)
	}
}

func TestSubmitHelpShowsLifecycleCommandsAndHidesDeprecatedCreate(t *testing.T) {
	stdout, stderr, runErr := runRootCommand(t, []string{"submit"})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, subcommand := range []string{"status", "cancel", "preflight"} {
		if !usageListsSubcommand(stderr, subcommand) {
			t.Fatalf("expected submit help to list %s, got %q", subcommand, stderr)
		}
	}
	if usageListsSubcommand(stderr, "create") {
		t.Fatalf("expected submit help to hide deprecated create path, got %q", stderr)
	}
	if !strings.Contains(stderr, "asc release run") {
		t.Fatalf("expected submit help to point App Store users to asc release run, got %q", stderr)
	}
}

func TestPublishAppStoreHelpShowsDeprecatedCompatibilityGuidance(t *testing.T) {
	usage := usageForCommand(t, "publish", "appstore")

	if !strings.Contains(usage, "DEPRECATED: use `asc release run`.") {
		t.Fatalf("expected deprecated guidance in publish appstore help, got %q", usage)
	}
	if !strings.Contains(usage, "Deprecated compatibility path") {
		t.Fatalf("expected compatibility guidance in publish appstore help, got %q", usage)
	}
	if strings.Contains(usage, "--ipa") {
		t.Fatalf("expected deprecated publish appstore help to hide legacy flag details, got %q", usage)
	}
}

func TestSubmitCreateHelpShowsDeprecatedCompatibilityGuidance(t *testing.T) {
	usage := usageForCommand(t, "submit", "create")

	if !strings.Contains(usage, "DEPRECATED: use `asc release run`.") {
		t.Fatalf("expected deprecated guidance in submit create help, got %q", usage)
	}
	if !strings.Contains(usage, "Deprecated compatibility path") {
		t.Fatalf("expected compatibility guidance in submit create help, got %q", usage)
	}
	if strings.Contains(usage, "--build") {
		t.Fatalf("expected deprecated submit create help to hide legacy flag details, got %q", usage)
	}
}

func TestDeprecatedPublishAppStoreInvocationWarns(t *testing.T) {
	stdout, stderr, runErr := runRootCommand(t, []string{"publish", "appstore", "--app", "app-1", "--version", "1.0.0"})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	requireStderrContainsWarning(t, stderr, publishAppStoreDeprecationWarning)
	if !strings.Contains(stderr, "Error: --ipa is required") {
		t.Fatalf("expected legacy validation error after deprecation warning, got %q", stderr)
	}
}

func TestDeprecatedSubmitCreateInvocationWarns(t *testing.T) {
	stdout, stderr, runErr := runRootCommand(t, []string{"submit", "create", "--version", "1.0.0", "--version-id", "version-1", "--build", "build-1", "--confirm"})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	requireStderrContainsWarning(t, stderr, submitCreateDeprecationWarning)
	if !strings.Contains(stderr, "--version and --version-id are mutually exclusive") {
		t.Fatalf("expected legacy validation error after deprecation warning, got %q", stderr)
	}
}
