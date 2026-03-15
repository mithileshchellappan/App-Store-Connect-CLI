package xcode

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestGetVersion_NotMacOS(t *testing.T) {
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := GetVersion(context.Background(), ".")
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestSetVersion_NotMacOS(t *testing.T) {
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := SetVersion(context.Background(), SetVersionOptions{ProjectDir: ".", Version: "1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestBumpVersion_NotMacOS(t *testing.T) {
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := BumpVersion(context.Background(), BumpVersionOptions{ProjectDir: ".", BumpType: BumpPatch})
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestGetVersion_MissingAgvtool(t *testing.T) {
	prev := lookPathFn
	lookPathFn = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}
	defer func() { lookPathFn = prev }()

	prevOS := runtimeGOOS
	runtimeGOOS = "darwin"
	defer func() { runtimeGOOS = prevOS }()

	_, err := GetVersion(context.Background(), ".")
	if err == nil || !strings.Contains(err.Error(), "agvtool") {
		t.Fatalf("expected agvtool not found error, got: %v", err)
	}
}

func TestBumpVersionType_Validate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"major", true},
		{"minor", true},
		{"patch", true},
		{"build", true},
		{"MAJOR", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		_, err := ParseBumpType(tt.input)
		if tt.valid && err != nil {
			t.Errorf("ParseBumpType(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ParseBumpType(%q) expected error", tt.input)
		}
	}
}

func TestIsVariableReference(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"$(MARKETING_VERSION)", true},
		{"1.2.3", false},
		{"$(CURRENT_PROJECT_VERSION)", true},
		{"", false},
	}

	for _, tt := range tests {
		if got := isVariableReference(tt.input); got != tt.want {
			t.Errorf("isVariableReference(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIncrementBuildString(t *testing.T) {
	tests := []struct {
		current string
		want    string
		wantErr bool
	}{
		{"42", "43", false},
		{"1", "2", false},
		{"1.2.3", "1.2.4", false},
		{"", "", true},
		{"abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			got, err := incrementBuildString(tt.current)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("incrementBuildString(%q) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

func TestBumpVersionString(t *testing.T) {
	tests := []struct {
		current  string
		bumpType BumpType
		want     string
		wantErr  bool
	}{
		{"1.2.3", BumpPatch, "1.2.4", false},
		{"1.2.3", BumpMinor, "1.3.0", false},
		{"1.2.3", BumpMajor, "2.0.0", false},
		{"1.0", BumpPatch, "", true},
		{"1.0", BumpMinor, "1.1", false},
		{"1.0", BumpMajor, "2.0", false},
		{"bad", BumpPatch, "", true},
		{"", BumpPatch, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+string(tt.bumpType), func(t *testing.T) {
			got, err := bumpVersionString(tt.current, tt.bumpType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bumpVersionString(%q, %s) = %q, want %q", tt.current, tt.bumpType, got, tt.want)
			}
		})
	}
}
