package main

import (
	"encoding/base64"
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rudrankriyam/App-Store-Connect-CLI/apps/studio/internal/studio/acp"
	"github.com/rudrankriyam/App-Store-Connect-CLI/apps/studio/internal/studio/approvals"
	"github.com/rudrankriyam/App-Store-Connect-CLI/apps/studio/internal/studio/settings"
	"github.com/rudrankriyam/App-Store-Connect-CLI/apps/studio/internal/studio/threads"
)

func TestParseAppsListOutputAcceptsEmptyEnvelope(t *testing.T) {
	rawApps, err := parseAppsListOutput([]byte(`{"data":[]}`))
	if err != nil {
		t.Fatalf("parseAppsListOutput() error = %v", err)
	}
	if len(rawApps) != 0 {
		t.Fatalf("len(rawApps) = %d, want 0", len(rawApps))
	}
}

func TestParseAvailabilityViewOutputReturnsResourceID(t *testing.T) {
	availabilityID, available, err := parseAvailabilityViewOutput([]byte(`{"data":{"id":"availability-123","attributes":{"availableInNewTerritories":true}}}`))
	if err != nil {
		t.Fatalf("parseAvailabilityViewOutput() error = %v", err)
	}
	if availabilityID != "availability-123" {
		t.Fatalf("availabilityID = %q, want availability-123", availabilityID)
	}
	if !available {
		t.Fatal("available = false, want true")
	}
}

func TestParseFirstAppPriceReference(t *testing.T) {
	priceID := base64.RawURLEncoding.EncodeToString([]byte(`{"t":"CAN","p":"price-point-42"}`))

	ref, found, err := parseFirstAppPriceReference([]byte(`{"data":[{"id":"` + priceID + `"}]}`))
	if err != nil {
		t.Fatalf("parseFirstAppPriceReference() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if ref.Territory != "CAN" || ref.PricePoint != "price-point-42" {
		t.Fatalf("ref = %+v, want territory CAN and price point price-point-42", ref)
	}
}

func TestParseAppPricePointLookupUsesIncludedTerritoryCurrency(t *testing.T) {
	pricePointID := base64.RawURLEncoding.EncodeToString([]byte(`{"p":"price-point-42"}`))

	result, found, err := parseAppPricePointLookup([]byte(`{
		"data":[{"id":"`+pricePointID+`","attributes":{"customerPrice":"4.99","proceeds":"3.49"}}],
		"included":[{"type":"territories","id":"CAN","attributes":{"currency":"CAD"}}]
	}`), "CAN", "price-point-42")
	if err != nil {
		t.Fatalf("parseAppPricePointLookup() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if result.Price != "4.99" || result.Proceeds != "3.49" || result.Currency != "CAD" {
		t.Fatalf("result = %+v, want price 4.99, proceeds 3.49, currency CAD", result)
	}
}

func TestParseASCCommandArgsSupportsQuotedValues(t *testing.T) {
	args, err := parseASCCommandArgs(`status --app "123 456" --output json`)
	if err != nil {
		t.Fatalf("parseASCCommandArgs() error = %v", err)
	}
	want := []string{"status", "--app", "123 456", "--output", "json"}
	if len(args) != len(want) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestParseASCCommandArgsRejectsUnterminatedQuotes(t *testing.T) {
	if _, err := parseASCCommandArgs(`status --app "123`); err == nil {
		t.Fatal("parseASCCommandArgs() error = nil, want error")
	}
}

func TestBundledASCPathPrefersAppBundleResources(t *testing.T) {
	tmp := t.TempDir()
	resourceDir := filepath.Join(tmp, "ASC Studio.app", "Contents", "Resources", "bin")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	bundled := filepath.Join(resourceDir, "asc")
	if err := os.WriteFile(bundled, []byte("binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalExecutable := osExecutableFunc
	originalGetwd := getwdFunc
	t.Cleanup(func() {
		osExecutableFunc = originalExecutable
		getwdFunc = originalGetwd
	})

	osExecutableFunc = func() (string, error) {
		return filepath.Join(tmp, "ASC Studio.app", "Contents", "MacOS", "ASC Studio"), nil
	}
	getwdFunc = func() (string, error) {
		return filepath.Join(tmp, "workspace"), nil
	}

	app := &App{}
	if got := app.bundledASCPath(); got != bundled {
		t.Fatalf("bundledASCPath() = %q, want %q", got, bundled)
	}
}

func TestConfigGuardSerializesConcurrentSnapshots(t *testing.T) {
	tmpHome := t.TempDir()
	configDir := filepath.Join(tmpHome, ".asc")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"profile":"good"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
	t.Cleanup(func() {
		if originalHome == "" {
			_ = os.Unsetenv("HOME")
			return
		}
		_ = os.Setenv("HOME", originalHome)
	})

	restoreFirst := configGuard()
	if err := os.WriteFile(configPath, []byte(`{"profile":"corrupted"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	secondReady := make(chan func(), 1)
	go func() {
		secondReady <- configGuard()
	}()

	select {
	case <-secondReady:
		t.Fatal("second configGuard() returned before the first restore unlocked it")
	case <-time.After(150 * time.Millisecond):
	}

	restoreFirst()

	var restoreSecond func()
	select {
	case restoreSecond = <-secondReady:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second configGuard()")
	}

	if err := os.WriteFile(configPath, []byte(`{"profile":"second-corruption"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	restoreSecond()

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != `{"profile":"good"}` {
		t.Fatalf("config contents = %q, want original snapshot", string(got))
	}
}

func TestEnsureSessionSingleFlightsConcurrentCalls(t *testing.T) {
	tmp := t.TempDir()
	settingsStore := settings.NewStore(tmp)
	if err := settingsStore.Save(settings.StudioSettings{
		AgentCommand:     "fake-agent",
		WorkspaceRoot:    "/tmp/workspace",
		PreferBundledASC: true,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var startCalls atomic.Int32
	started := make(chan struct{}, 1)
	release := make(chan struct{})

	app := &App{
		rootDir:      tmp,
		settings:     settingsStore,
		threads:      threads.NewStore(tmp),
		approvals:    approvals.NewQueue(),
		sessions:     make(map[string]*threadSession),
		sessionInits: make(map[string]chan struct{}),
		startAgent: func(context.Context, acp.LaunchSpec) (agentClient, error) {
			startCalls.Add(1)
			return &fakeAgentClient{
				bootstrapFn: func(context.Context, acp.SessionConfig) (string, error) {
					select {
					case started <- struct{}{}:
					default:
					}
					<-release
					return "session-1", nil
				},
			}, nil
		},
	}

	thread := threads.Thread{ID: "thread-1"}
	results := make(chan *threadSession, 2)
	errorsCh := make(chan error, 2)

	go func() {
		session, err := app.ensureSession(thread)
		errorsCh <- err
		results <- session
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first agent bootstrap")
	}

	go func() {
		session, err := app.ensureSession(thread)
		errorsCh <- err
		results <- session
	}()

	close(release)

	var sessions []*threadSession
	for range 2 {
		select {
		case err := <-errorsCh:
			if err != nil {
				t.Fatalf("ensureSession() error = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for ensureSession result")
		}
		select {
		case session := <-results:
			sessions = append(sessions, session)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for session")
		}
	}

	if got := startCalls.Load(); got != 1 {
		t.Fatalf("startCalls = %d, want 1", got)
	}
	if len(sessions) != 2 || sessions[0] != sessions[1] {
		t.Fatal("ensureSession() did not reuse the same session for concurrent callers")
	}
}

type fakeAgentClient struct {
	bootstrapFn func(context.Context, acp.SessionConfig) (string, error)
	promptFn    func(context.Context, string, string) (acp.PromptResult, []acp.Event, error)
	closeFn     func() error
}

func (f *fakeAgentClient) Bootstrap(ctx context.Context, cfg acp.SessionConfig) (string, error) {
	if f.bootstrapFn != nil {
		return f.bootstrapFn(ctx, cfg)
	}
	return "session-1", nil
}

func (f *fakeAgentClient) Prompt(ctx context.Context, sessionID string, prompt string) (acp.PromptResult, []acp.Event, error) {
	if f.promptFn != nil {
		return f.promptFn(ctx, sessionID, prompt)
	}
	return acp.PromptResult{Status: "completed"}, nil, nil
}

func (f *fakeAgentClient) Close() error {
	if f.closeFn != nil {
		return f.closeFn()
	}
	return nil
}
