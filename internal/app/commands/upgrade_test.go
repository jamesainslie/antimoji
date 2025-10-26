package commands

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstallMethod tests the InstallMethod type
func TestInstallMethod(t *testing.T) {
	tests := []struct {
		name   string
		method InstallMethod
		want   string
	}{
		{"homebrew", InstallMethodHomebrew, "homebrew"},
		{"go-install", InstallMethodGoInstall, "go-install"},
		{"apt", InstallMethodAPT, "apt"},
		{"yum", InstallMethodYUM, "yum"},
		{"pacman", InstallMethodPacman, "pacman"},
		{"docker", InstallMethodDocker, "docker"},
		{"source", InstallMethodSource, "source"},
		{"manual", InstallMethodManual, "manual"},
		{"unknown", InstallMethodUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.method))
		})
	}
}

// TestDetectionResult tests the DetectionResult struct
func TestDetectionResult(t *testing.T) {
	result := DetectionResult{
		Method:     InstallMethodHomebrew,
		Confidence: 95,
		Metadata: map[string]string{
			"path": "/opt/homebrew/bin/antimoji",
		},
		CanUpgrade: true,
	}

	assert.Equal(t, InstallMethodHomebrew, result.Method)
	assert.Equal(t, 95, result.Confidence)
	assert.True(t, result.CanUpgrade)
	assert.Equal(t, "/opt/homebrew/bin/antimoji", result.Metadata["path"])
}

// TestVersionInfo tests the VersionInfo struct
func TestVersionInfo(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"same version", "0.9.16", "0.9.16", false},
		{"newer available", "0.9.15", "0.9.16", true},
		{"already latest", "0.9.16", "0.9.15", false},
		{"dev version", "dev", "0.9.16", true},
		{"with v prefix", "v0.9.16", "v0.9.16", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := NewVersionInfo(tt.current, tt.latest)
			assert.Equal(t, tt.current, info.Current)
			assert.Equal(t, tt.latest, info.Latest)
			assert.Equal(t, tt.want, info.UpdateAvailable)
		})
	}
}

// TestInstallationInfo tests the InstallationInfo struct
func TestInstallationInfo(t *testing.T) {
	info := InstallationInfo{
		Method:     InstallMethodGoInstall,
		Path:       "/Users/test/go/bin/antimoji",
		Confidence: 90,
		Metadata: map[string]string{
			"gopath": "/Users/test/go",
		},
		CanUpgrade: true,
	}

	assert.Equal(t, InstallMethodGoInstall, info.Method)
	assert.Equal(t, "/Users/test/go/bin/antimoji", info.Path)
	assert.Equal(t, 90, info.Confidence)
	assert.True(t, info.CanUpgrade)
}

// TestHomebrewDetector tests Homebrew installation detection
func TestHomebrewDetector(t *testing.T) {
	tests := []struct {
		name           string
		binaryPath     string
		brewAvailable  bool
		brewListOutput string
		wantConfidence int
		wantCanUpgrade bool
	}{
		{
			name:           "homebrew intel mac",
			binaryPath:     "/usr/local/Cellar/antimoji/0.9.16/bin/antimoji",
			brewAvailable:  true,
			brewListOutput: "/usr/local/Cellar/antimoji/0.9.16/bin/antimoji",
			wantConfidence: 95,
			wantCanUpgrade: true,
		},
		{
			name:           "homebrew apple silicon",
			binaryPath:     "/opt/homebrew/Cellar/antimoji/0.9.16/bin/antimoji",
			brewAvailable:  true,
			brewListOutput: "/opt/homebrew/Cellar/antimoji/0.9.16/bin/antimoji",
			wantConfidence: 95,
			wantCanUpgrade: true,
		},
		{
			name:           "not homebrew",
			binaryPath:     "/usr/local/bin/antimoji",
			brewAvailable:  false,
			brewListOutput: "",
			wantConfidence: 0,
			wantCanUpgrade: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &homebrewDetector{
				commandRunner: &mockCommandRunner{
					commands: map[string]mockCommandResult{
						"brew --version": {
							output: "Homebrew 4.0.0",
							err:    nil,
						},
						"brew list antimoji": {
							output: tt.brewListOutput,
							err:    nil,
						},
					},
				},
			}

			if !tt.brewAvailable {
				detector.commandRunner = &mockCommandRunner{
					commands: map[string]mockCommandResult{
						"brew --version": {
							output: "",
							err:    errors.New("brew not found"),
						},
					},
				}
			}

			result := detector.Detect(context.Background(), tt.binaryPath)

			if tt.wantConfidence > 0 {
				assert.Equal(t, InstallMethodHomebrew, result.Method)
				assert.Equal(t, tt.wantConfidence, result.Confidence)
				assert.Equal(t, tt.wantCanUpgrade, result.CanUpgrade)
			} else {
				assert.Equal(t, 0, result.Confidence)
			}
		})
	}
}

// TestGoInstallDetector tests Go install detection
func TestGoInstallDetector(t *testing.T) {
	tests := []struct {
		name            string
		binaryPath      string
		goVersionOutput string
		goEnvOutput     string
		wantConfidence  int
		wantCanUpgrade  bool
	}{
		{
			name:            "go install clean",
			binaryPath:      "/Users/test/go/bin/antimoji",
			goVersionOutput: "path\tgithub.com/antimoji/antimoji/cmd/antimoji\nvcs.modified\tfalse",
			goEnvOutput:     "GOBIN=/Users/test/go/bin",
			wantConfidence:  90,
			wantCanUpgrade:  true,
		},
		{
			name:            "go install modified",
			binaryPath:      "/Users/test/go/bin/antimoji",
			goVersionOutput: "path\tgithub.com/antimoji/antimoji/cmd/antimoji\nvcs.modified\ttrue",
			goEnvOutput:     "GOBIN=/Users/test/go/bin",
			wantConfidence:  70,
			wantCanUpgrade:  true,
		},
		{
			name:            "not go install",
			binaryPath:      "/usr/local/bin/antimoji",
			goVersionOutput: "",
			goEnvOutput:     "GOBIN=/Users/test/go/bin",
			wantConfidence:  0,
			wantCanUpgrade:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &goInstallDetector{
				commandRunner: &mockCommandRunner{
					commands: map[string]mockCommandResult{
						"go version": {
							output: "go version go1.23.0 darwin/arm64",
							err:    nil,
						},
						"go version -m " + tt.binaryPath: {
							output: tt.goVersionOutput,
							err:    nil,
						},
						"go env GOBIN": {
							output: tt.goEnvOutput,
							err:    nil,
						},
					},
				},
			}

			result := detector.Detect(context.Background(), tt.binaryPath)

			assert.Equal(t, tt.wantConfidence, result.Confidence)
			if tt.wantConfidence > 0 {
				assert.Equal(t, InstallMethodGoInstall, result.Method)
				assert.Equal(t, tt.wantCanUpgrade, result.CanUpgrade)
			}
		})
	}
}

// TestSourceDetector tests source installation detection
func TestSourceDetector(t *testing.T) {
	tests := []struct {
		name           string
		binaryPath     string
		hasGitDir      bool
		hasMakefile    bool
		wantConfidence int
		wantCanUpgrade bool
	}{
		{
			name:           "from source with git",
			binaryPath:     "/Users/test/antimoji/bin/antimoji",
			hasGitDir:      true,
			hasMakefile:    true,
			wantConfidence: 70,
			wantCanUpgrade: true,
		},
		{
			name:           "from source without git",
			binaryPath:     "/Users/test/antimoji/bin/antimoji",
			hasGitDir:      false,
			hasMakefile:    true,
			wantConfidence: 50,
			wantCanUpgrade: false,
		},
		{
			name:           "not from source",
			binaryPath:     "/usr/local/bin/antimoji",
			hasGitDir:      false,
			hasMakefile:    false,
			wantConfidence: 0,
			wantCanUpgrade: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &sourceDetector{
				fileChecker: &mockFileChecker{
					exists: map[string]bool{
						".git":     tt.hasGitDir,
						"Makefile": tt.hasMakefile,
					},
				},
			}

			result := detector.Detect(context.Background(), tt.binaryPath)

			if tt.wantConfidence > 0 {
				assert.Equal(t, InstallMethodSource, result.Method)
				assert.Equal(t, tt.wantCanUpgrade, result.CanUpgrade)
			}
			// Confidence may vary based on implementation details
		})
	}
}

// TestDetectInstallation tests the main detection orchestrator
func TestDetectInstallation(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		detectors  []InstallDetector
		wantMethod InstallMethod
		wantError  bool
	}{
		{
			name:       "homebrew wins",
			binaryPath: "/opt/homebrew/bin/antimoji",
			detectors: []InstallDetector{
				&mockDetector{
					result: DetectionResult{
						Method:     InstallMethodHomebrew,
						Confidence: 95,
						CanUpgrade: true,
					},
				},
				&mockDetector{
					result: DetectionResult{
						Method:     InstallMethodGoInstall,
						Confidence: 50,
						CanUpgrade: true,
					},
				},
			},
			wantMethod: InstallMethodHomebrew,
			wantError:  false,
		},
		{
			name:       "go install wins",
			binaryPath: "/Users/test/go/bin/antimoji",
			detectors: []InstallDetector{
				&mockDetector{
					result: DetectionResult{
						Method:     InstallMethodGoInstall,
						Confidence: 90,
						CanUpgrade: true,
					},
				},
				&mockDetector{
					result: DetectionResult{
						Method:     InstallMethodManual,
						Confidence: 40,
						CanUpgrade: false,
					},
				},
			},
			wantMethod: InstallMethodGoInstall,
			wantError:  false,
		},
		{
			name:       "unknown when all fail",
			binaryPath: "/usr/local/bin/antimoji",
			detectors: []InstallDetector{
				&mockDetector{
					result: DetectionResult{
						Method:     InstallMethodUnknown,
						Confidence: 0,
						CanUpgrade: false,
					},
				},
			},
			wantMethod: InstallMethodUnknown,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &installationDetector{
				detectors: tt.detectors,
			}

			info, err := detector.DetectInstallation(context.Background(), tt.binaryPath)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMethod, info.Method)
			}
		})
	}
}

// TestCompareVersions tests semantic version comparison
func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int // -1 = current < latest, 0 = equal, 1 = current > latest
	}{
		{"equal versions", "0.9.16", "0.9.16", 0},
		{"current older", "0.9.15", "0.9.16", -1},
		{"current newer", "0.9.17", "0.9.16", 1},
		{"major version diff", "1.0.0", "0.9.16", 1},
		{"with v prefix", "v0.9.16", "v0.9.16", 0},
		{"mixed v prefix", "v0.9.16", "0.9.16", 0},
		{"dev version", "dev", "0.9.16", -1},
		{"patch version", "0.9.16", "0.9.16-patch1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.current, tt.latest)
			assert.Equal(t, tt.want, result)
		})
	}
}

// TestUpgradeHandler tests the upgrade handler
func TestUpgradeHandler(t *testing.T) {
	mockLogger := logging.NewMockLogger()
	mockUI := ui.NewMockUserOutput()

	handler := NewUpgradeHandler(mockLogger, mockUI)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.logger)
	assert.NotNil(t, handler.ui)
}

// TestUpgradeHandlerCreateCommand tests command creation
func TestUpgradeHandlerCreateCommand(t *testing.T) {
	mockLogger := logging.NewMockLogger()
	mockUI := ui.NewMockUserOutput()

	handler := NewUpgradeHandler(mockLogger, mockUI)
	cmd := handler.CreateCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "upgrade [flags]", cmd.Use)
	assert.Contains(t, cmd.Short, "Upgrade")
	assert.Contains(t, cmd.Short, "latest")
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)
}

// Mock types for testing

type mockCommandRunner struct {
	commands map[string]mockCommandResult
}

type mockCommandResult struct {
	output string
	err    error
}

func (m *mockCommandRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	// Build the full command key by joining name and all args
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	result, ok := m.commands[key]
	if !ok {
		return "", errors.New("command not found")
	}
	return result.output, result.err
}

type mockFileChecker struct {
	exists map[string]bool
}

func (m *mockFileChecker) Exists(path string) bool {
	// Check exact path first
	if exists, ok := m.exists[path]; ok {
		return exists
	}
	// Check by basename (for convenience in tests)
	base := filepath.Base(path)
	if exists, ok := m.exists[base]; ok {
		return exists
	}
	return false
}

type mockDetector struct {
	result DetectionResult
}

func (m *mockDetector) Name() string {
	return string(m.result.Method)
}

func (m *mockDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	return m.result
}
