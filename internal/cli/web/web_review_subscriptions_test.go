package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
	webcore "github.com/rudrankriyam/App-Store-Connect-CLI/internal/web"
)

func TestWebReviewSubscriptionsListCommandOutputsJSON(t *testing.T) {
	_ = stubWebProgressLabels(t)

	origResolveSession := resolveSessionFn
	t.Cleanup(func() { resolveSessionFn = origResolveSession })

	resolveSessionFn = func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet {
						t.Fatalf("unexpected method: %s", req.Method)
					}
					if req.URL.Path != "/iris/v1/apps/app-1/subscriptionGroups" {
						t.Fatalf("unexpected path: %s", req.URL.Path)
					}
					body := `{
						"data": [{
							"id": "group-1",
							"type": "subscriptionGroups",
							"attributes": {"referenceName": "Premium"},
							"relationships": {
								"subscriptions": {"data": [{"type": "subscriptions", "id": "sub-1"}]}
							}
						}],
						"included": [{
							"id": "sub-1",
							"type": "subscriptions",
							"attributes": {
								"productId": "com.example.monthly",
								"name": "Monthly",
								"state": "READY_TO_SUBMIT",
								"isAppStoreReviewInProgress": false,
								"submitWithNextAppStoreVersion": true
							}
						}]
					}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(body)),
						Request:    req,
					}, nil
				}),
			},
		}, "cache", nil
	}

	cmd := WebReviewSubscriptionsListCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--app", "app-1",
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, _ := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("exec error: %v", err)
		}
	})

	var payload reviewSubscriptionsListOutput
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse stdout JSON: %v\nstdout=%s", err, stdout)
	}
	if payload.AppID != "app-1" {
		t.Fatalf("expected app-1, got %#v", payload)
	}
	if payload.AttachedCount != 1 || len(payload.Subscriptions) != 1 {
		t.Fatalf("unexpected list output: %#v", payload)
	}
	if payload.Subscriptions[0].ID != "sub-1" || !payload.Subscriptions[0].SubmitWithNextAppStoreVersion {
		t.Fatalf("unexpected subscription output: %#v", payload.Subscriptions[0])
	}
}

func TestWebReviewSubscriptionsAttachCommandRefreshesState(t *testing.T) {
	labels := stubWebProgressLabels(t)

	origResolveSession := resolveSessionFn
	t.Cleanup(func() { resolveSessionFn = origResolveSession })

	listCalls := 0
	resolveSessionFn = func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					switch {
					case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/app-1/subscriptionGroups":
						listCalls++
						attached := "false"
						if listCalls > 1 {
							attached = "true"
						}
						body := `{
							"data": [{
								"id": "group-1",
								"type": "subscriptionGroups",
								"attributes": {"referenceName": "Premium"},
								"relationships": {
									"subscriptions": {"data": [{"type": "subscriptions", "id": "sub-1"}]}
								}
							}],
							"included": [{
								"id": "sub-1",
								"type": "subscriptions",
								"attributes": {
									"productId": "com.example.monthly",
									"name": "Monthly",
									"state": "READY_TO_SUBMIT",
									"isAppStoreReviewInProgress": false,
									"submitWithNextAppStoreVersion": ` + attached + `
								}
							}]
						}`
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader(body)),
							Request:    req,
						}, nil

					case req.Method == http.MethodPost && req.URL.Path == "/iris/v1/subscriptionSubmissions":
						body := `{
							"data": {
								"id": "submission-1",
								"type": "subscriptionSubmissions",
								"attributes": {"submitWithNextAppStoreVersion": true},
								"relationships": {
									"subscription": {"data": {"type": "subscriptions", "id": "sub-1"}}
								}
							}
						}`
						return &http.Response{
							StatusCode: http.StatusCreated,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader(body)),
							Request:    req,
						}, nil

					default:
						t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
						return nil, nil
					}
				}),
			},
		}, "cache", nil
	}

	cmd := WebReviewSubscriptionsAttachCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--app", "app-1",
		"--subscription-id", "sub-1",
		"--confirm",
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, _ := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("exec error: %v", err)
		}
	})

	var payload reviewSubscriptionMutationOutput
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse stdout JSON: %v\nstdout=%s", err, stdout)
	}
	if payload.Operation != "attach" || !payload.Changed || payload.SubmissionID != "submission-1" {
		t.Fatalf("unexpected attach output: %#v", payload)
	}
	if !payload.Subscription.SubmitWithNextAppStoreVersion {
		t.Fatalf("expected refreshed attached subscription, got %#v", payload.Subscription)
	}
	if listCalls != 2 {
		t.Fatalf("expected list before and after attach, got %d calls", listCalls)
	}
	wantLabels := []string{
		"Loading review subscriptions",
		"Attaching subscription to next app version",
		"Refreshing review subscriptions",
	}
	if strings.Join(*labels, "|") != strings.Join(wantLabels, "|") {
		t.Fatalf("expected labels %v, got %v", wantLabels, *labels)
	}
}

func TestWebReviewSubscriptionsRemoveCommandRefreshesState(t *testing.T) {
	_ = stubWebProgressLabels(t)

	origResolveSession := resolveSessionFn
	t.Cleanup(func() { resolveSessionFn = origResolveSession })

	listCalls := 0
	resolveSessionFn = func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					switch {
					case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/app-1/subscriptionGroups":
						listCalls++
						attached := "true"
						if listCalls > 1 {
							attached = "false"
						}
						body := `{
							"data": [{
								"id": "group-1",
								"type": "subscriptionGroups",
								"attributes": {"referenceName": "Premium"},
								"relationships": {
									"subscriptions": {"data": [{"type": "subscriptions", "id": "sub-1"}]}
								}
							}],
							"included": [{
								"id": "sub-1",
								"type": "subscriptions",
								"attributes": {
									"productId": "com.example.monthly",
									"name": "Monthly",
									"state": "READY_TO_SUBMIT",
									"isAppStoreReviewInProgress": false,
									"submitWithNextAppStoreVersion": ` + attached + `
								}
							}]
						}`
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader(body)),
							Request:    req,
						}, nil

					case req.Method == http.MethodDelete && req.URL.Path == "/iris/v1/subscriptionSubmissions/sub-1":
						return &http.Response{
							StatusCode: http.StatusNoContent,
							Header:     http.Header{"Content-Type": []string{"application/json"}},
							Body:       io.NopCloser(strings.NewReader("")),
							Request:    req,
						}, nil

					default:
						t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
						return nil, nil
					}
				}),
			},
		}, "cache", nil
	}

	cmd := WebReviewSubscriptionsRemoveCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--app", "app-1",
		"--subscription-id", "sub-1",
		"--confirm",
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, _ := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("exec error: %v", err)
		}
	})

	var payload reviewSubscriptionMutationOutput
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse stdout JSON: %v\nstdout=%s", err, stdout)
	}
	if payload.Operation != "remove" || !payload.Changed {
		t.Fatalf("unexpected remove output: %#v", payload)
	}
	if payload.Subscription.SubmitWithNextAppStoreVersion {
		t.Fatalf("expected refreshed detached subscription, got %#v", payload.Subscription)
	}
	if listCalls != 2 {
		t.Fatalf("expected list before and after remove, got %d calls", listCalls)
	}
}

func TestWebReviewSubscriptionsAttachRequiresConfirm(t *testing.T) {
	cmd := WebReviewSubscriptionsAttachCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--app", "app-1",
		"--subscription-id", "sub-1",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	_, stderr := captureOutput(t, func() {
		err := cmd.Exec(context.Background(), nil)
		if err == nil {
			t.Fatal("expected missing confirm error")
		}
	})
	if !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("expected confirm guidance in stderr, got %q", stderr)
	}
}

func TestWebReviewSubscriptionsAttachFailsFastForMissingMetadata(t *testing.T) {
	labels := stubWebProgressLabels(t)

	origResolveSession := resolveSessionFn
	t.Cleanup(func() { resolveSessionFn = origResolveSession })

	resolveSessionFn = func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet || req.URL.Path != "/iris/v1/apps/app-1/subscriptionGroups" {
						t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
					}
					body := `{
						"data": [{
							"id": "group-1",
							"type": "subscriptionGroups",
							"attributes": {"referenceName": "Premium"},
							"relationships": {
								"subscriptions": {"data": [{"type": "subscriptions", "id": "sub-1"}]}
							}
						}],
						"included": [{
							"id": "sub-1",
							"type": "subscriptions",
							"attributes": {
								"productId": "com.example.monthly",
								"name": "Monthly",
								"state": "MISSING_METADATA",
								"isAppStoreReviewInProgress": false,
								"submitWithNextAppStoreVersion": false
							}
						}]
					}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(body)),
						Request:    req,
					}, nil
				}),
			},
		}, "cache", nil
	}

	cmd := WebReviewSubscriptionsAttachCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--app", "app-1",
		"--subscription-id", "sub-1",
		"--confirm",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	_, stderr := captureOutput(t, func() {
		err := cmd.Exec(context.Background(), nil)
		if err == nil {
			t.Fatal("expected missing-metadata preflight error")
		}
		var reported shared.ReportedError
		if !errors.As(err, &reported) {
			t.Fatalf("expected ReportedError, got %T: %v", err, err)
		}
	})

	if !strings.Contains(stderr, "is MISSING_METADATA") {
		t.Fatalf("expected missing metadata preflight explanation, got %q", stderr)
	}
	if !strings.Contains(stderr, `asc validate subscriptions --app "app-1"`) {
		t.Fatalf("expected validate subscriptions hint, got %q", stderr)
	}
	if !strings.Contains(stderr, `asc subscriptions images create --subscription-id "sub-1" --file "./image.png"`) {
		t.Fatalf("expected promotional image hint, got %q", stderr)
	}
	wantLabels := []string{"Loading review subscriptions"}
	if strings.Join(*labels, "|") != strings.Join(wantLabels, "|") {
		t.Fatalf("expected labels %v, got %v", wantLabels, *labels)
	}
}
