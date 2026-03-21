package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/itunes"
)

func runCommand(t *testing.T, args []string) (string, string, error) {
	t.Helper()

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse(args); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	return stdout, stderr, runErr
}

func TestAppsHelpShowsPublicSubcommand(t *testing.T) {
	root := RootCommand("1.2.3")
	var appsCmd any
	for _, sub := range root.Subcommands {
		if sub != nil && sub.Name == "apps" {
			appsCmd = sub
			break
		}
	}
	if appsCmd == nil {
		t.Fatal("expected apps command in root subcommands")
	}

	usage := appsCmd.(*ffcli.Command).UsageFunc(appsCmd.(*ffcli.Command))
	if !strings.Contains(usage, "public") {
		t.Fatalf("expected apps help to show public subcommand, got %q", usage)
	}
}

func TestAppsPublicHelpShowsSubcommands(t *testing.T) {
	root := RootCommand("1.2.3")
	publicCmd := findSubcommand(root, "apps", "public")
	if publicCmd == nil {
		t.Fatal("expected apps public command")
	}

	usage := publicCmd.UsageFunc(publicCmd)
	for _, want := range []string{"view", "search", "prices", "descriptions", "storefronts", "No authentication is required."} {
		if !strings.Contains(usage, want) {
			t.Fatalf("expected apps public help to contain %q, got %q", want, usage)
		}
	}
}

func TestAppsPublicValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "view missing app",
			args:    []string{"apps", "public", "view"},
			wantErr: "--app is required",
		},
		{
			name:    "view conflicting app aliases",
			args:    []string{"apps", "public", "view", "--app", "123", "--id", "456"},
			wantErr: "--app and --id are mutually exclusive",
		},
		{
			name:    "view invalid app id",
			args:    []string{"apps", "public", "view", "--app", "abc"},
			wantErr: "--app must be a numeric App Store app ID",
		},
		{
			name:    "view invalid country",
			args:    []string{"apps", "public", "view", "--app", "123", "--country", "zz"},
			wantErr: "unsupported country code",
		},
		{
			name:    "search missing term",
			args:    []string{"apps", "public", "search"},
			wantErr: "--term is required",
		},
		{
			name:    "search invalid limit",
			args:    []string{"apps", "public", "search", "--term", "focus", "--limit", "0"},
			wantErr: "--limit must be between 1 and 200",
		},
		{
			name:    "search invalid country",
			args:    []string{"apps", "public", "search", "--term", "focus", "--country", "zz"},
			wantErr: "unsupported country code",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, runErr := runCommand(t, test.args)
			if !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("expected ErrHelp, got %v", runErr)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected stderr to contain %q, got %q", test.wantErr, stderr)
			}
		})
	}
}

func TestAppsPublicAliasIsSilentAndMatchesCanonical(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_KEY_ID", "poison")
	t.Setenv("ASC_ISSUER_ID", "poison")
	t.Setenv("ASC_PRIVATE_KEY_PATH", "/nonexistent")
	t.Setenv("ASC_PRIVATE_KEY", "poison")
	t.Setenv("ASC_PRIVATE_KEY_B64", "poison")

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	lookupBody := `{
		"resultCount": 1,
		"results": [{
			"trackId": 123,
			"trackName": "Alpha",
			"bundleId": "com.example.alpha",
			"trackViewUrl": "https://apps.apple.com/us/app/alpha/id123",
			"artworkUrl512": "https://example.com/icon.png",
			"sellerName": "Alpha Inc",
			"primaryGenreName": "Games",
			"genres": ["Games", "Action"],
			"version": "1.0.0",
			"description": "Alpha description",
			"price": 0,
			"formattedPrice": "Free",
			"currency": "USD",
			"averageUserRating": 4.5,
			"userRatingCount": 12,
			"averageUserRatingForCurrentVersion": 4.4,
			"userRatingCountForCurrentVersion": 11
		}]
	}`

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/lookup" {
			t.Fatalf("expected /lookup, got %s", req.URL.Path)
		}
		if got := req.URL.Query().Get("id"); got != "123" {
			t.Fatalf("expected id=123, got %q", got)
		}
		if got := req.URL.Query().Get("country"); got != "us" {
			t.Fatalf("expected country=us, got %q", got)
		}
		if got := req.URL.Query().Get("entity"); got != "software" {
			t.Fatalf("expected entity=software, got %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(lookupBody)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	canonicalStdout, canonicalStderr, canonicalErr := runCommand(t, []string{"apps", "public", "view", "--app", "123", "--output", "json"})
	aliasStdout, aliasStderr, aliasErr := runCommand(t, []string{"apps", "public", "view", "--id", "123", "--output", "json"})
	matchingStdout, matchingStderr, matchingErr := runCommand(t, []string{"apps", "public", "view", "--app", "123", "--id", "123", "--output", "json"})

	if canonicalErr != nil {
		t.Fatalf("canonical run error: %v", canonicalErr)
	}
	if aliasErr != nil {
		t.Fatalf("alias run error: %v", aliasErr)
	}
	if matchingErr != nil {
		t.Fatalf("matching alias run error: %v", matchingErr)
	}
	if canonicalStderr != "" {
		t.Fatalf("expected canonical stderr to be empty, got %q", canonicalStderr)
	}
	if aliasStderr != "" {
		t.Fatalf("expected alias stderr to be empty, got %q", aliasStderr)
	}
	if matchingStderr != "" {
		t.Fatalf("expected matching alias stderr to be empty, got %q", matchingStderr)
	}
	if canonicalStdout != aliasStdout {
		t.Fatalf("expected canonical and alias outputs to match, canonical=%q alias=%q", canonicalStdout, aliasStdout)
	}
	if canonicalStdout != matchingStdout {
		t.Fatalf("expected canonical and matching alias outputs to match, canonical=%q matching=%q", canonicalStdout, matchingStdout)
	}

	var payload itunes.App
	if err := json.Unmarshal([]byte(canonicalStdout), &payload); err != nil {
		t.Fatalf("unmarshal view payload: %v", err)
	}
	if payload.Country != "US" {
		t.Fatalf("Country = %q, want US", payload.Country)
	}
	if payload.CountryName != "United States" {
		t.Fatalf("CountryName = %q, want United States", payload.CountryName)
	}
}
