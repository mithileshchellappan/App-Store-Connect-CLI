package cmdtest

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildsListPlatformFilter(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		// When --platform is provided, the CLI should use the /v1/builds endpoint
		// with filter[preReleaseVersion.platform]=TV_OS
		if req.URL.Path != "/v1/builds" {
			t.Fatalf("expected /v1/builds path for platform filter, got %q", req.URL.Path)
		}
		query := req.URL.Query()
		if query.Get("filter[preReleaseVersion.platform]") != "TV_OS" {
			t.Fatalf("expected filter[preReleaseVersion.platform]=TV_OS, got %q", query.Get("filter[preReleaseVersion.platform]"))
		}
		if query.Get("filter[app]") != "123456789" {
			t.Fatalf("expected filter[app]=123456789, got %q", query.Get("filter[app]"))
		}
		body := `{"data":[{"type":"builds","id":"build-tvos-1","attributes":{"version":"9","uploadedDate":"2026-03-13T00:00:00Z"}}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"builds", "list", "--app", "123456789", "--platform", "TV_OS"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"build-tvos-1"`) {
		t.Fatalf("expected build output, got %q", stdout)
	}
}

func TestBuildsListPlatformFilterCaseInsensitive(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		query := req.URL.Query()
		if query.Get("filter[preReleaseVersion.platform]") != "IOS" {
			t.Fatalf("expected normalized platform IOS, got %q", query.Get("filter[preReleaseVersion.platform]"))
		}
		body := `{"data":[{"type":"builds","id":"build-ios-1"}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"builds", "list", "--app", "123456789", "--platform", "ios"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if !strings.Contains(stdout, `"id":"build-ios-1"`) {
		t.Fatalf("expected build output, got %q", stdout)
	}
}

func TestBuildsListPlatformFilterRejectsInvalid(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	captureOutput(t, func() {
		if err := root.Parse([]string{"builds", "list", "--app", "123456789", "--platform", "ANDROID"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr == nil {
		t.Fatal("expected platform validation error")
	}
	if !strings.Contains(runErr.Error(), "--platform must be one of") {
		t.Fatalf("expected platform validation error, got %v", runErr)
	}
}
