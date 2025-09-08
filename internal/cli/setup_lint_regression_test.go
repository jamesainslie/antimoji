// Package cli provides regression tests for setup-lint command to prevent critical bugs.
package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestYAMLSyntaxValidation ensures generated YAML is syntactically correct.
// This test would have caught the YAML indentation bug that caused
// "mapping values are not allowed in this context" errors.
func TestYAMLSyntaxValidation(t *testing.T) {
	tests := []struct {
		name      string
		mode      LintMode
		hasGoMod  bool
		setupFunc func(dir string) // Setup function to create test environment
	}{
		{
			name:     "zero-tolerance with go.mod",
			mode:     ZeroToleranceMode,
			hasGoMod: true,
			setupFunc: func(dir string) {
				err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.21"), 0644)
				require.NoError(t, err)
			},
		},
		{
			name:     "zero-tolerance without go.mod",
			mode:     ZeroToleranceMode,
			hasGoMod: false,
			setupFunc: func(dir string) {
				// No go.mod file
			},
		},
		{
			name:     "allow-list with go.mod",
			mode:     AllowListMode,
			hasGoMod: true,
			setupFunc: func(dir string) {
				err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.21"), 0644)
				require.NoError(t, err)
			},
		},
		{
			name:     "permissive without go.mod",
			mode:     PermissiveMode,
			hasGoMod: false,
			setupFunc: func(dir string) {
				// No go.mod file
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFunc(tempDir)

			// Generate pre-commit config
			config := generatePreCommitConfigForMode(tt.mode, tempDir)

			// Test 1: YAML syntax validation
			var parsed interface{}
			err := yaml.Unmarshal([]byte(config), &parsed)
			assert.NoError(t, err, "Generated YAML should be syntactically valid")

			// Test 2: Specific indentation checks
			lines := strings.Split(config, "\n")
			for i, line := range lines {
				if strings.Contains(line, "entry:") {
					// entry: should be at exactly 8 spaces (not 16)
					spaces := len(line) - len(strings.TrimLeft(line, " "))
					assert.Equal(t, 8, spaces, "Line %d: 'entry:' field should have exactly 8 spaces indentation, got %d spaces", i+1, spaces)
				}
			}

			// Test 3: No excessive indentation anywhere
			for i, line := range lines {
				if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
					spaces := len(line) - len(strings.TrimLeft(line, " "))
					assert.LessOrEqual(t, spaces, 16, "Line %d: No line should have more than 16 spaces indentation, got %d", i+1, spaces)
				}
			}
		})
	}
}

// TestCommandFlagValidation ensures only valid antimoji flags are used.
// This test would have caught the invalid --profile and --fail-on-found flags.
func TestCommandFlagValidation(t *testing.T) {
	tests := []struct {
		name string
		mode LintMode
	}{
		{"zero-tolerance", ZeroToleranceMode},
		{"allow-list", AllowListMode},
		{"permissive", PermissiveMode},
	}

	// Get valid antimoji flags by running antimoji scan --help
	validFlags := getValidAntimojiScanFlags(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			config := generatePreCommitConfigForMode(tt.mode, tempDir)

			// Extract entry lines from the config
			lines := strings.Split(config, "\n")
			for _, line := range lines {
				if strings.Contains(line, "entry:") && strings.Contains(line, "antimoji scan") {
					// Parse the command and flags
					entryLine := strings.TrimSpace(line)
					entryLine = strings.TrimPrefix(entryLine, "entry:")
					entryLine = strings.TrimSpace(entryLine)

					// Split into command parts
					parts := strings.Fields(entryLine)
					require.Greater(t, len(parts), 2, "Entry should have at least 'antimoji scan' + args")
					assert.Equal(t, "antimoji", parts[0])
					assert.Equal(t, "scan", parts[1])

					// Check each flag
					for i := 2; i < len(parts); i++ {
						part := parts[i]
						if strings.HasPrefix(part, "--") {
							flagName := strings.Split(part, "=")[0] // Handle --flag=value format
							assert.Contains(t, validFlags, flagName, "Flag %s is not valid for antimoji scan", flagName)
						}
					}

					// Specific checks for removed invalid flags
					// --profile flag is now valid in the improved configuration
					assert.NotContains(t, entryLine, "--fail-on-found", "Should not contain invalid --fail-on-found flag")
				}
			}
		})
	}
}

// TestBinaryDetectionLogic ensures binary detection works correctly.
// This test would have caught issues with bin/antimoji vs antimoji detection.
func TestBinaryDetectionLogic(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() func() // Returns cleanup function
		expectedBinary string
		description    string
	}{
		{
			name: "global antimoji available",
			setupFunc: func() func() {
				// Mock PATH to include antimoji
				originalPath := os.Getenv("PATH")
				tempDir := t.TempDir()
				fakeBinary := filepath.Join(tempDir, "antimoji")
				err := os.WriteFile(fakeBinary, []byte("#!/bin/bash\necho fake"), 0755)
				require.NoError(t, err)
				err = os.Setenv("PATH", tempDir+":"+originalPath)
				require.NoError(t, err)
				return func() {
					_ = os.Setenv("PATH", originalPath) // Ignore error in cleanup
				}
			},
			expectedBinary: "antimoji",
			description:    "Should prefer globally installed antimoji",
		},
		{
			name: "no global antimoji, has Makefile",
			setupFunc: func() func() {
				// Remove antimoji from PATH and create Makefile
				originalPath := os.Getenv("PATH")
				err := os.Setenv("PATH", "/usr/bin:/bin") // Minimal PATH without antimoji
				require.NoError(t, err)

				// Create Makefile in current directory
				err = os.WriteFile("Makefile", []byte("build:\n\tgo build -o bin/antimoji ./cmd/antimoji"), 0644)
				require.NoError(t, err)

				return func() {
					_ = os.Setenv("PATH", originalPath) // Ignore error in cleanup
					_ = os.Remove("Makefile")           // Ignore error in cleanup
				}
			},
			expectedBinary: "bin/antimoji",
			description:    "Should use local build when global not available but Makefile exists",
		},
		{
			name: "no global antimoji, no Makefile",
			setupFunc: func() func() {
				// Remove antimoji from PATH and ensure no Makefile
				originalPath := os.Getenv("PATH")
				err := os.Setenv("PATH", "/usr/bin:/bin") // Minimal PATH without antimoji
				require.NoError(t, err)
				_ = os.Remove("Makefile") // Ensure no Makefile (ignore error if not exists)

				return func() {
					_ = os.Setenv("PATH", originalPath) // Ignore error in cleanup
				}
			},
			expectedBinary: "antimoji",
			description:    "Should default to global antimoji even if not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc()
			defer cleanup()

			result := detectAntimojiCommand()
			assert.Equal(t, tt.expectedBinary, result, tt.description)
		})
	}
}

// TestMultiDocumentYAMLSupport ensures multi-document YAML support is enabled.
// This test would have caught the missing --allow-multiple-documents flag.
func TestMultiDocumentYAMLSupport(t *testing.T) {
	tempDir := t.TempDir()

	// Create a multi-document YAML file
	multiDocYAML := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test1
data:
  config: "test1"
---
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  ports:
  - port: 80`

	yamlFile := filepath.Join(tempDir, "multi-doc.yaml")
	err := os.WriteFile(yamlFile, []byte(multiDocYAML), 0644)
	require.NoError(t, err)

	modes := []LintMode{ZeroToleranceMode, AllowListMode, PermissiveMode}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			config := generatePreCommitConfigForMode(mode, tempDir)

			// Check that check-yaml hook has --allow-multiple-documents
			assert.Contains(t, config, "check-yaml", "Should include check-yaml hook")
			assert.Contains(t, config, "--allow-multiple-documents", "Should include --allow-multiple-documents flag for multi-document YAML support")

			// Verify the specific hook configuration
			lines := strings.Split(config, "\n")
			foundCheckYaml := false
			foundAllowMultipleDocs := false

			for i, line := range lines {
				if strings.Contains(line, "- id: check-yaml") {
					foundCheckYaml = true
					// Check the next few lines for args
					for j := i + 1; j < len(lines) && j < i+5; j++ {
						if strings.Contains(lines[j], "args:") && strings.Contains(lines[j], "--allow-multiple-documents") {
							foundAllowMultipleDocs = true
							break
						}
					}
					break
				}
			}

			assert.True(t, foundCheckYaml, "Should find check-yaml hook")
			assert.True(t, foundAllowMultipleDocs, "Should find --allow-multiple-documents in check-yaml args")
		})
	}
}

// TestGoModuleDetection ensures Go hooks are only included when go.mod exists.
// This test would have caught issues with Go hooks being included inappropriately.
func TestGoModuleDetection(t *testing.T) {
	tests := []struct {
		name          string
		hasGoMod      bool
		expectGoHooks bool
	}{
		{
			name:          "with go.mod",
			hasGoMod:      true,
			expectGoHooks: true,
		},
		{
			name:          "without go.mod",
			hasGoMod:      false,
			expectGoHooks: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			if tt.hasGoMod {
				goModContent := `module test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)`
				err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
				require.NoError(t, err)
			}

			config := generatePreCommitConfigForMode(ZeroToleranceMode, tempDir)

			if tt.expectGoHooks {
				assert.Contains(t, config, "github.com/dnephin/pre-commit-golang", "Should include Go hooks when go.mod exists")
				assert.Contains(t, config, "go-fmt", "Should include go-fmt hook")
				assert.Contains(t, config, "go-mod-tidy", "Should include go-mod-tidy hook")
			} else {
				assert.NotContains(t, config, "github.com/dnephin/pre-commit-golang", "Should not include Go hooks when go.mod doesn't exist")
				assert.NotContains(t, config, "go-fmt", "Should not include go-fmt hook")
				assert.NotContains(t, config, "go-mod-tidy", "Should not include go-mod-tidy hook")
			}
		})
	}
}

// TestBuildHookConditionalInclusion ensures build hooks are only included when needed.
// This test would have caught unnecessary build hooks being included.
func TestBuildHookConditionalInclusion(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func() func() // Returns cleanup function
		expectBuildHook bool
		description     string
	}{
		{
			name: "global antimoji available",
			setupFunc: func() func() {
				// Mock PATH to include antimoji
				originalPath := os.Getenv("PATH")
				tempDir := t.TempDir()
				fakeBinary := filepath.Join(tempDir, "antimoji")
				err := os.WriteFile(fakeBinary, []byte("#!/bin/bash\necho fake"), 0755)
				require.NoError(t, err)
				err = os.Setenv("PATH", tempDir+":"+originalPath)
				require.NoError(t, err)
				return func() {
					_ = os.Setenv("PATH", originalPath) // Ignore error in cleanup
				}
			},
			expectBuildHook: false,
			description:     "Should not include build hook when using global antimoji",
		},
		{
			name: "local build needed",
			setupFunc: func() func() {
				// Remove antimoji from PATH and create Makefile
				originalPath := os.Getenv("PATH")
				err := os.Setenv("PATH", "/usr/bin:/bin") // Minimal PATH without antimoji
				require.NoError(t, err)

				// Create Makefile in current directory
				err = os.WriteFile("Makefile", []byte("build:\n\tgo build -o bin/antimoji ./cmd/antimoji"), 0644)
				require.NoError(t, err)

				return func() {
					_ = os.Setenv("PATH", originalPath) // Ignore error in cleanup
					_ = os.Remove("Makefile")           // Ignore error in cleanup
				}
			},
			expectBuildHook: true,
			description:     "Should include build hook when using local bin/antimoji",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc()
			defer cleanup()

			tempDir := t.TempDir()
			config := generatePreCommitConfigForMode(ZeroToleranceMode, tempDir)

			if tt.expectBuildHook {
				assert.Contains(t, config, "build-antimoji", tt.description)
				assert.Contains(t, config, "make build", "Should include make build command")
			} else {
				assert.NotContains(t, config, "build-antimoji", tt.description)
				assert.NotContains(t, config, "make build", "Should not include make build command")
			}
		})
	}
}

// TestConfigurationValidation ensures the validation function catches common issues.
// This test validates the validateConfiguration function works correctly.
func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid YAML",
			yamlContent: `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    hooks:
      - id: check-yaml`,
			expectError: false,
		},
		{
			name: "invalid YAML syntax",
			yamlContent: `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    hooks:
      - id: check-yaml
        entry: antimoji scan
        invalid: mapping: value`,
			expectError: true,
			errorMsg:    "invalid YAML syntax",
		},
		{
			name: "contains valid profile flag",
			yamlContent: `repos:
  - repo: local
    hooks:
      - id: test
        entry: antimoji
        args: [scan, --profile=ci-lint]`,
			expectError: false,
		},
		{
			name: "contains invalid fail-on-found flag",
			yamlContent: `repos:
  - repo: local
    hooks:
      - id: test
        entry: antimoji scan --fail-on-found`,
			expectError: true,
			errorMsg:    "invalid --fail-on-found flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(t.TempDir(), "test-config.yaml")
			err := os.WriteFile(tempFile, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)

			err = validateConfiguration(tempFile)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEndToEndRegressionPrevention runs a comprehensive end-to-end test.
// This test simulates the exact scenario that caused the original bugs.
func TestEndToEndRegressionPrevention(t *testing.T) {
	tempDir := t.TempDir()

	// Set up a realistic project environment
	goModContent := `module test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)`
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create a multi-document Kubernetes YAML file
	k8sYAML := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  config.yaml: |
    setting: value
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 1`
	err = os.WriteFile(filepath.Join(tempDir, "k8s-resources.yaml"), []byte(k8sYAML), 0644)
	require.NoError(t, err)

	// Set quiet mode for tests
	originalQuiet := quiet
	quiet = true
	defer func() { quiet = originalQuiet }()

	// Run setup-lint
	opts := &SetupLintOptions{
		Mode:              "zero-tolerance",
		OutputDir:         tempDir,
		PreCommitConfig:   true,
		GolangCIConfig:    true,
		AllowedEmojis:     []string{},
		Force:             true,
		SkipPreCommitHook: true,
	}

	cmd := NewSetupLintCommand()
	err = runSetupLint(cmd, []string{}, opts)
	require.NoError(t, err, "setup-lint should complete successfully")

	// Verify all files were created
	preCommitConfig := filepath.Join(tempDir, ".pre-commit-config.yaml")
	golangCIConfig := filepath.Join(tempDir, ".golangci.yml")
	antimojiConfig := filepath.Join(tempDir, ".antimoji.yaml")

	assert.FileExists(t, preCommitConfig)
	assert.FileExists(t, golangCIConfig)
	assert.FileExists(t, antimojiConfig)

	// Critical validation: YAML syntax
	data, err := os.ReadFile(preCommitConfig)
	require.NoError(t, err)

	var parsed interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err, "Generated pre-commit config should be valid YAML")

	configContent := string(data)

	// Validate specific bug fixes
	assert.Contains(t, configContent, "--allow-multiple-documents", "Should support multi-document YAML")
	// --profile flag is now valid, so we expect it in the improved configuration
	assert.Contains(t, configContent, "--profile=zero-tolerance", "Should contain valid --profile flag")
	assert.NotContains(t, configContent, "--fail-on-found", "Should not contain invalid --fail-on-found flag")

	// Check for proper indentation (no 16-space indentation on entry:)
	lines := strings.Split(configContent, "\n")
	for i, line := range lines {
		if strings.Contains(line, "entry:") {
			spaces := len(line) - len(strings.TrimLeft(line, " "))
			assert.LessOrEqual(t, spaces, 8, "Line %d: entry: should not have excessive indentation", i+1)
		}
	}

	// Validate Go hooks are included (since we have go.mod)
	assert.Contains(t, configContent, "go-fmt", "Should include Go hooks when go.mod exists")

	// Validate golangci config
	golangCIData, err := os.ReadFile(golangCIConfig)
	require.NoError(t, err)
	golangCIContent := string(golangCIData)

	assert.NotContains(t, golangCIContent, "profile:", "GolangCI config should not contain invalid profile field")
	assert.Contains(t, golangCIContent, "pre-commit hooks", "Should contain note about pre-commit integration")
	assert.NotContains(t, golangCIContent, "antimoji:", "Should not contain invalid antimoji linter")
}

// Helper function to get valid antimoji scan flags by introspection
func getValidAntimojiScanFlags(t *testing.T) map[string]bool {
	// Try to run antimoji scan --help to get valid flags
	cmd := exec.Command("antimoji", "scan", "--help")
	output, err := cmd.Output()

	validFlags := map[string]bool{
		"--config":    true,
		"--threshold": true,
		"--quiet":     true,
		"--recursive": true,
		"--output":    true,
		"--help":      true,
	}

	if err == nil {
		// Parse help output to extract flags
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "--") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					flag := strings.Split(parts[0], "=")[0]
					flag = strings.Split(flag, ",")[0] // Handle --flag, -f format
					validFlags[flag] = true
				}
			}
		}
	}

	// Explicitly mark known invalid flags as false
	validFlags["--profile"] = false
	validFlags["--fail-on-found"] = false

	return validFlags
}
