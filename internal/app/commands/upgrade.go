// Package commands provides CLI command implementations using dependency injection.
package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/antimoji/antimoji/internal/observability/logging"
	"github.com/antimoji/antimoji/internal/ui"
	"github.com/spf13/cobra"
)

// InstallMethod represents how antimoji was installed
type InstallMethod string

const (
	// InstallMethodHomebrew indicates installation via Homebrew
	InstallMethodHomebrew InstallMethod = "homebrew"
	// InstallMethodGoInstall indicates installation via go install
	InstallMethodGoInstall InstallMethod = "go-install"
	// InstallMethodAPT indicates installation via APT/DEB package manager
	InstallMethodAPT InstallMethod = "apt"
	// InstallMethodYUM indicates installation via YUM/RPM package manager
	InstallMethodYUM InstallMethod = "yum"
	// InstallMethodPacman indicates installation via Pacman package manager
	InstallMethodPacman InstallMethod = "pacman"
	// InstallMethodDocker indicates running in a container
	InstallMethodDocker InstallMethod = "docker"
	// InstallMethodSource indicates installation from source
	InstallMethodSource InstallMethod = "source"
	// InstallMethodManual indicates manual binary installation
	InstallMethodManual InstallMethod = "manual"
	// InstallMethodUnknown indicates unknown installation method
	InstallMethodUnknown InstallMethod = "unknown"
)

// DetectionResult represents the result of an installation method detection
type DetectionResult struct {
	Method     InstallMethod
	Confidence int               // 0-100
	Metadata   map[string]string // Additional metadata about the detection
	CanUpgrade bool              // Whether automatic upgrade is possible
}

// VersionInfo contains version information
type VersionInfo struct {
	Current         string
	Latest          string
	UpdateAvailable bool
}

// InstallationInfo contains information about the installation
type InstallationInfo struct {
	Method     InstallMethod
	Path       string
	Confidence int
	Metadata   map[string]string
	CanUpgrade bool
}

// UpgradeOptions holds the options for the upgrade command
type UpgradeOptions struct {
	CheckOnly bool // Only check for updates, don't upgrade
	Force     bool // Force upgrade even if on latest version
}

// InstallDetector defines the interface for installation method detectors
type InstallDetector interface {
	Name() string
	Detect(ctx context.Context, binaryPath string) DetectionResult
}

// CommandRunner defines the interface for running shell commands
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// FileChecker defines the interface for checking file existence
type FileChecker interface {
	Exists(path string) bool
}

// UpgradeHandler handles the upgrade command with dependency injection
type UpgradeHandler struct {
	logger    logging.Logger
	ui        ui.UserOutput
	version   string
	detector  *installationDetector
	upgrader  *upgradeExecutor
	apiClient *githubAPIClient
}

// NewUpgradeHandler creates a new upgrade command handler
func NewUpgradeHandler(logger logging.Logger, uiOutput ui.UserOutput) *UpgradeHandler {
	cmdRunner := &defaultCommandRunner{}
	fileChecker := &defaultFileChecker{}

	return &UpgradeHandler{
		logger:  logger,
		ui:      uiOutput,
		version: "0.9.16", // Will be set from build info
		detector: &installationDetector{
			detectors: []InstallDetector{
				&homebrewDetector{commandRunner: cmdRunner},
				&goInstallDetector{commandRunner: cmdRunner},
				&aptDetector{commandRunner: cmdRunner},
				&yumDetector{commandRunner: cmdRunner},
				&pacmanDetector{commandRunner: cmdRunner},
				&dockerDetector{fileChecker: fileChecker},
				&sourceDetector{fileChecker: fileChecker, commandRunner: cmdRunner},
				&manualDetector{},
			},
		},
		upgrader: &upgradeExecutor{
			commandRunner: cmdRunner,
		},
		apiClient: &githubAPIClient{
			httpClient: &http.Client{Timeout: 10 * time.Second},
		},
	}
}

// SetVersion sets the current version for the handler
func (h *UpgradeHandler) SetVersion(version string) {
	h.version = version
}

// CreateCommand creates the upgrade cobra command
func (h *UpgradeHandler) CreateCommand() *cobra.Command {
	opts := &UpgradeOptions{}

	cmd := &cobra.Command{
		Use:   "upgrade [flags]",
		Short: "Upgrade antimoji to the latest version",
		Long: `Upgrade antimoji to the latest version.

This command detects how antimoji was installed and uses the appropriate
method to upgrade to the latest version from GitHub releases.

Supported installation methods:
  - Homebrew (macOS/Linux)
  - Go install (cross-platform)
  - APT/DEB (Debian/Ubuntu)
  - YUM/RPM (RedHat/Fedora/CentOS)
  - Pacman (Arch Linux)
  - From source (git + make)

Examples:
  antimoji upgrade                 # Upgrade to latest version
  antimoji upgrade --check-only    # Check for updates without upgrading
  antimoji upgrade --force         # Force upgrade even if on latest`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Execute(cmd.Context(), cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.CheckOnly, "check-only", false, "only check for updates without upgrading")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "force upgrade even if already on latest version")

	return cmd
}

// Execute runs the upgrade command
func (h *UpgradeHandler) Execute(ctx context.Context, cmd *cobra.Command, args []string, opts *UpgradeOptions) error {
	h.logger.Info(ctx, "Starting upgrade command",
		"check_only", opts.CheckOnly,
		"force", opts.Force,
	)

	// Get executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Detect installation method
	h.ui.Info(ctx, "Detecting installation method...")
	installInfo, err := h.detector.DetectInstallation(ctx, exePath)
	if err != nil {
		return fmt.Errorf("failed to detect installation method: %w", err)
	}

	h.ui.Info(ctx, "Detected installation method: %s (confidence: %d%%)", installInfo.Method, installInfo.Confidence)
	h.logger.Info(ctx, "Installation method detected",
		"method", string(installInfo.Method),
		"confidence", installInfo.Confidence,
		"can_upgrade", installInfo.CanUpgrade,
		"path", installInfo.Path,
	)

	// Check for updates
	h.ui.Info(ctx, "Checking for updates...")
	versionInfo, err := h.apiClient.CheckForUpdates(ctx, h.version)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	h.ui.Info(ctx, "Current version: %s", versionInfo.Current)
	h.ui.Info(ctx, "Latest version:  %s", versionInfo.Latest)

	if !versionInfo.UpdateAvailable && !opts.Force {
		h.ui.Success(ctx, "You are already running the latest version!")
		return nil
	}

	if !versionInfo.UpdateAvailable && opts.Force {
		h.ui.Info(ctx, "Forcing upgrade even though you're on the latest version...")
	}

	if opts.CheckOnly {
		if versionInfo.UpdateAvailable {
			h.ui.Info(ctx, "Update available: %s -> %s", versionInfo.Current, versionInfo.Latest)
			return nil
		}
		return nil
	}

	// Check if automatic upgrade is possible
	if !installInfo.CanUpgrade {
		h.ui.Warning(ctx, "Automatic upgrade is not available for this installation method.")
		h.showManualUpgradeInstructions(ctx, installInfo.Method)
		return nil
	}

	// Perform upgrade
	h.ui.Info(ctx, "Upgrading antimoji from %s to %s...", versionInfo.Current, versionInfo.Latest)
	if err := h.upgrader.ExecuteUpgrade(ctx, installInfo, h.ui); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	h.ui.Success(ctx, "Successfully upgraded to version %s!", versionInfo.Latest)
	h.ui.Info(ctx, "Run 'antimoji version' to verify the upgrade.")

	return nil
}

// showManualUpgradeInstructions displays manual upgrade instructions
func (h *UpgradeHandler) showManualUpgradeInstructions(ctx context.Context, method InstallMethod) {
	h.ui.Info(ctx, "\nManual upgrade instructions:")
	h.ui.Info(ctx, "")

	switch method {
	case InstallMethodDocker:
		h.ui.Info(ctx, "  Pull the latest Docker image:")
		h.ui.Info(ctx, "    docker pull ghcr.io/jamesainslie/antimoji:latest")
	case InstallMethodSource:
		h.ui.Info(ctx, "  Navigate to your source directory and update:")
		h.ui.Info(ctx, "    cd /path/to/antimoji")
		h.ui.Info(ctx, "    git pull origin main")
		h.ui.Info(ctx, "    make build")
		h.ui.Info(ctx, "    sudo make install")
	case InstallMethodManual:
		h.ui.Info(ctx, "  Download the latest release from GitHub:")
		h.ui.Info(ctx, "    https://github.com/jamesainslie/antimoji/releases/latest")
		h.ui.Info(ctx, "  Or use one of the package managers:")
		h.ui.Info(ctx, "    brew install antimoji")
		h.ui.Info(ctx, "    go install github.com/antimoji/antimoji/cmd/antimoji@latest")
	default:
		h.ui.Info(ctx, "  Visit the GitHub releases page:")
		h.ui.Info(ctx, "    https://github.com/jamesainslie/antimoji/releases/latest")
	}
}

// NewVersionInfo creates a VersionInfo with computed UpdateAvailable field
func NewVersionInfo(current, latest string) VersionInfo {
	updateAvailable := compareVersions(current, latest) < 0
	return VersionInfo{
		Current:         current,
		Latest:          latest,
		UpdateAvailable: updateAvailable,
	}
}

// compareVersions compares two semantic versions
// Returns: -1 if v1 < v2, 0 if equal, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Strip 'v' prefix
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Handle dev version
	if v1 == "dev" || v1 == "unknown" {
		return -1
	}
	if v2 == "dev" || v2 == "unknown" {
		return 1
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		// Extract numeric part
		num1 := extractNumber(parts1[i])
		num2 := extractNumber(parts2[i])

		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}

		// If numbers are equal, check for suffixes (e.g., "16" vs "16-patch1")
		// Version with suffix is considered newer
		hasSuffix1 := strings.ContainsAny(parts1[i], "-+")
		hasSuffix2 := strings.ContainsAny(parts2[i], "-+")

		if !hasSuffix1 && hasSuffix2 {
			// v1 has no suffix, v2 has suffix -> v1 < v2
			return -1
		}
		if hasSuffix1 && !hasSuffix2 {
			// v1 has suffix, v2 has no suffix -> v1 > v2
			return 1
		}
		// If both have suffixes or neither have suffixes, continue to next part
	}

	// If all parts equal, longer version is considered newer
	if len(parts1) < len(parts2) {
		return -1
	}
	if len(parts1) > len(parts2) {
		return 1
	}

	return 0
}

// extractNumber extracts the numeric part from a version component
func extractNumber(s string) int {
	// Handle pre-release versions (e.g., "16-patch1" -> 16)
	if idx := strings.IndexAny(s, "-+"); idx != -1 {
		s = s[:idx]
	}

	num, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return num
}

// installationDetector orchestrates multiple detectors
type installationDetector struct {
	detectors []InstallDetector
}

// DetectInstallation runs all detectors and returns the best result
func (d *installationDetector) DetectInstallation(ctx context.Context, binaryPath string) (InstallationInfo, error) {
	type result struct {
		info DetectionResult
		err  error
	}

	results := make(chan result, len(d.detectors))

	// Run all detectors in parallel
	for _, detector := range d.detectors {
		go func(det InstallDetector) {
			info := det.Detect(ctx, binaryPath)
			results <- result{info: info, err: nil}
		}(detector)
	}

	// Collect results
	var detectionResults []DetectionResult
	for i := 0; i < len(d.detectors); i++ {
		res := <-results
		if res.err == nil && res.info.Confidence > 40 {
			detectionResults = append(detectionResults, res.info)
		}
	}

	// Find best result
	if len(detectionResults) == 0 {
		return InstallationInfo{
			Method:     InstallMethodUnknown,
			Path:       binaryPath,
			Confidence: 0,
			Metadata:   make(map[string]string),
			CanUpgrade: false,
		}, nil
	}

	// Sort by confidence, then by priority
	best := detectionResults[0]
	for _, r := range detectionResults[1:] {
		if r.Confidence > best.Confidence {
			best = r
		} else if r.Confidence == best.Confidence && r.CanUpgrade && !best.CanUpgrade {
			best = r
		}
	}

	return InstallationInfo{
		Method:     best.Method,
		Path:       binaryPath,
		Confidence: best.Confidence,
		Metadata:   best.Metadata,
		CanUpgrade: best.CanUpgrade,
	}, nil
}

// Default implementations

type defaultCommandRunner struct{}

func (r *defaultCommandRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

type defaultFileChecker struct{}

func (c *defaultFileChecker) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Homebrew detector

type homebrewDetector struct {
	commandRunner CommandRunner
}

func (d *homebrewDetector) Name() string {
	return "homebrew"
}

func (d *homebrewDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	confidence := 0
	metadata := make(map[string]string)

	// Check if binary is in Homebrew path
	if strings.Contains(binaryPath, "/Cellar/antimoji") {
		confidence += 50
		metadata["cellar_path"] = binaryPath
	}

	// Check if brew command is available
	if _, err := d.commandRunner.Run(ctx, "brew", "--version"); err == nil {
		confidence += 20

		// Verify antimoji is installed via brew
		if output, err := d.commandRunner.Run(ctx, "brew", "list", "antimoji"); err == nil && strings.Contains(output, "antimoji") {
			confidence += 25
			metadata["brew_managed"] = "true"
		}
	}

	return DetectionResult{
		Method:     InstallMethodHomebrew,
		Confidence: confidence,
		Metadata:   metadata,
		CanUpgrade: confidence >= 70,
	}
}

// Go install detector

type goInstallDetector struct {
	commandRunner CommandRunner
}

func (d *goInstallDetector) Name() string {
	return "go-install"
}

func (d *goInstallDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	confidence := 0
	metadata := make(map[string]string)

	// Check if go command is available
	if _, err := d.commandRunner.Run(ctx, "go", "version"); err != nil {
		return DetectionResult{
			Method:     InstallMethodGoInstall,
			Confidence: 0,
			Metadata:   metadata,
			CanUpgrade: false,
		}
	}

	// Check build info - this is the primary indicator
	if output, err := d.commandRunner.Run(ctx, "go", "version", "-m", binaryPath); err == nil {
		if strings.Contains(output, "github.com/antimoji/antimoji/cmd/antimoji") {
			confidence += 70
			metadata["go_module"] = "true"

			if strings.Contains(output, "vcs.modified\tfalse") || !strings.Contains(output, "vcs.modified") {
				confidence += 20
			}
		}
	}

	// Check if in GOPATH/bin or GOBIN
	if gobinOutput, err := d.commandRunner.Run(ctx, "go", "env", "GOBIN"); err == nil {
		gobin := strings.TrimSpace(gobinOutput)
		if gobin != "" && strings.Contains(binaryPath, gobin) {
			confidence += 10
		}
	}

	return DetectionResult{
		Method:     InstallMethodGoInstall,
		Confidence: confidence,
		Metadata:   metadata,
		CanUpgrade: confidence >= 70,
	}
}

// APT detector

type aptDetector struct {
	commandRunner CommandRunner
}

func (d *aptDetector) Name() string {
	return "apt"
}

func (d *aptDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	confidence := 0
	metadata := make(map[string]string)

	// Check if dpkg is available
	if _, err := d.commandRunner.Run(ctx, "dpkg", "--version"); err != nil {
		return DetectionResult{
			Method:     InstallMethodAPT,
			Confidence: 0,
			Metadata:   metadata,
			CanUpgrade: false,
		}
	}

	// Check if antimoji is installed via dpkg
	if output, err := d.commandRunner.Run(ctx, "dpkg", "-S", binaryPath); err == nil && strings.Contains(output, "antimoji") {
		confidence = 95
		metadata["package_manager"] = "dpkg"
	}

	return DetectionResult{
		Method:     InstallMethodAPT,
		Confidence: confidence,
		Metadata:   metadata,
		CanUpgrade: confidence >= 90,
	}
}

// YUM detector

type yumDetector struct {
	commandRunner CommandRunner
}

func (d *yumDetector) Name() string {
	return "yum"
}

func (d *yumDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	confidence := 0
	metadata := make(map[string]string)

	// Check if rpm is available
	if output, err := d.commandRunner.Run(ctx, "rpm", "-qf", binaryPath); err == nil && strings.Contains(output, "antimoji") {
		confidence = 95
		metadata["package_manager"] = "rpm"
	}

	return DetectionResult{
		Method:     InstallMethodYUM,
		Confidence: confidence,
		Metadata:   metadata,
		CanUpgrade: confidence >= 90,
	}
}

// Pacman detector

type pacmanDetector struct {
	commandRunner CommandRunner
}

func (d *pacmanDetector) Name() string {
	return "pacman"
}

func (d *pacmanDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	confidence := 0
	metadata := make(map[string]string)

	// Check if pacman is available and owns the binary
	if output, err := d.commandRunner.Run(ctx, "pacman", "-Qo", binaryPath); err == nil && strings.Contains(output, "antimoji") {
		confidence = 95
		metadata["package_manager"] = "pacman"
	}

	return DetectionResult{
		Method:     InstallMethodPacman,
		Confidence: confidence,
		Metadata:   metadata,
		CanUpgrade: confidence >= 90,
	}
}

// Docker detector

type dockerDetector struct {
	fileChecker FileChecker
}

func (d *dockerDetector) Name() string {
	return "docker"
}

func (d *dockerDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	confidence := 0
	metadata := make(map[string]string)

	// Check for Docker environment markers
	if d.fileChecker.Exists("/.dockerenv") {
		confidence = 80
		metadata["container"] = "docker"
	}

	// Check Kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		confidence = 80
		metadata["container"] = "kubernetes"
	}

	return DetectionResult{
		Method:     InstallMethodDocker,
		Confidence: confidence,
		Metadata:   metadata,
		CanUpgrade: false, // Cannot auto-upgrade in container
	}
}

// Source detector

type sourceDetector struct {
	fileChecker   FileChecker
	commandRunner CommandRunner
}

func (d *sourceDetector) Name() string {
	return "source"
}

func (d *sourceDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	confidence := 0
	metadata := make(map[string]string)

	// Check if binary is in ./bin/ or similar
	if strings.Contains(binaryPath, "/bin/antimoji") && !strings.Contains(binaryPath, "/usr/") {
		// Look for .git directory in parent
		dir := filepath.Dir(filepath.Dir(binaryPath))
		gitPath := filepath.Join(dir, ".git")
		if d.fileChecker.Exists(gitPath) {
			confidence += 40
			metadata["git_repo"] = "true"
		}

		// Look for Makefile
		makefilePath := filepath.Join(dir, "Makefile")
		if d.fileChecker.Exists(makefilePath) {
			confidence += 30
			metadata["has_makefile"] = "true"
		}
	}

	return DetectionResult{
		Method:     InstallMethodSource,
		Confidence: confidence,
		Metadata:   metadata,
		CanUpgrade: confidence >= 60,
	}
}

// Manual detector (fallback)

type manualDetector struct{}

func (d *manualDetector) Name() string {
	return "manual"
}

func (d *manualDetector) Detect(ctx context.Context, binaryPath string) DetectionResult {
	// Manual installation has low confidence but is always possible
	return DetectionResult{
		Method:     InstallMethodManual,
		Confidence: 30,
		Metadata:   map[string]string{"fallback": "true"},
		CanUpgrade: false,
	}
}

// upgradeExecutor handles the actual upgrade process

type upgradeExecutor struct {
	commandRunner CommandRunner
}

// ExecuteUpgrade performs the upgrade based on installation method
func (e *upgradeExecutor) ExecuteUpgrade(ctx context.Context, info InstallationInfo, uiOutput ui.UserOutput) error {
	switch info.Method {
	case InstallMethodHomebrew:
		return e.upgradeHomebrew(ctx, uiOutput)
	case InstallMethodGoInstall:
		return e.upgradeGoInstall(ctx, uiOutput)
	case InstallMethodAPT:
		return e.upgradeAPT(ctx, uiOutput)
	case InstallMethodYUM:
		return e.upgradeYUM(ctx, uiOutput)
	case InstallMethodPacman:
		return e.upgradePacman(ctx, uiOutput)
	case InstallMethodSource:
		return e.upgradeSource(ctx, info, uiOutput)
	default:
		return errors.New("automatic upgrade not supported for this installation method")
	}
}

func (e *upgradeExecutor) upgradeHomebrew(ctx context.Context, uiOutput ui.UserOutput) error {
	uiOutput.Info(ctx, "Running: brew upgrade antimoji")
	output, err := e.commandRunner.Run(ctx, "brew", "upgrade", "antimoji")
	if err != nil {
		return fmt.Errorf("brew upgrade failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func (e *upgradeExecutor) upgradeGoInstall(ctx context.Context, uiOutput ui.UserOutput) error {
	uiOutput.Info(ctx, "Running: go install github.com/antimoji/antimoji/cmd/antimoji@latest")
	output, err := e.commandRunner.Run(ctx, "go", "install", "github.com/antimoji/antimoji/cmd/antimoji@latest")
	if err != nil {
		return fmt.Errorf("go install failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func (e *upgradeExecutor) upgradeAPT(ctx context.Context, uiOutput ui.UserOutput) error {
	uiOutput.Info(ctx, "Running: sudo apt update && sudo apt upgrade antimoji")
	if _, err := e.commandRunner.Run(ctx, "sudo", "apt", "update"); err != nil {
		return fmt.Errorf("apt update failed: %w", err)
	}
	if _, err := e.commandRunner.Run(ctx, "sudo", "apt", "upgrade", "-y", "antimoji"); err != nil {
		return fmt.Errorf("apt upgrade failed: %w", err)
	}
	return nil
}

func (e *upgradeExecutor) upgradeYUM(ctx context.Context, uiOutput ui.UserOutput) error {
	uiOutput.Info(ctx, "Running: sudo yum update antimoji")
	_, err := e.commandRunner.Run(ctx, "sudo", "yum", "update", "-y", "antimoji")
	if err != nil {
		// Try dnf if yum fails
		output, err := e.commandRunner.Run(ctx, "sudo", "dnf", "update", "-y", "antimoji")
		if err != nil {
			return fmt.Errorf("yum/dnf update failed: %w\nOutput: %s", err, output)
		}
	}
	return nil
}

func (e *upgradeExecutor) upgradePacman(ctx context.Context, uiOutput ui.UserOutput) error {
	uiOutput.Info(ctx, "Running: sudo pacman -Syu antimoji")
	output, err := e.commandRunner.Run(ctx, "sudo", "pacman", "-Syu", "--noconfirm", "antimoji")
	if err != nil {
		return fmt.Errorf("pacman upgrade failed: %w\nOutput: %s", err, output)
	}
	return nil
}

func (e *upgradeExecutor) upgradeSource(ctx context.Context, info InstallationInfo, uiOutput ui.UserOutput) error {
	if info.Metadata["git_repo"] != "true" {
		return errors.New("source installation without git repository cannot be auto-upgraded")
	}

	dir := filepath.Dir(filepath.Dir(info.Path))
	uiOutput.Info(ctx, "Updating source at: %s", dir)

	// Change to repo directory and pull
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() {
		_ = os.Chdir(currentDir) // Best effort to restore directory
	}()

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change to source directory: %w", err)
	}

	uiOutput.Info(ctx, "Running: git pull origin main")
	if _, err := e.commandRunner.Run(ctx, "git", "pull", "origin", "main"); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	uiOutput.Info(ctx, "Running: make build")
	if _, err := e.commandRunner.Run(ctx, "make", "build"); err != nil {
		return fmt.Errorf("make build failed: %w", err)
	}

	return nil
}

// githubAPIClient handles GitHub API interactions

type githubAPIClient struct {
	httpClient *http.Client
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
}

// CheckForUpdates checks GitHub for the latest release
func (c *githubAPIClient) CheckForUpdates(ctx context.Context, currentVersion string) (VersionInfo, error) {
	const apiURL = "https://api.github.com/repos/jamesainslie/antimoji/releases/latest"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Best effort to close response body
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return VersionInfo{}, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return VersionInfo{}, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	return NewVersionInfo(currentVersion, release.TagName), nil
}
