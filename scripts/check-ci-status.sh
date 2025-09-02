#!/bin/bash
# CI/CD Status Check Script for Antimoji

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Antimoji CI/CD Status Check${NC}"
echo "==========================="

# Check if we're in the right directory
if [ ! -f "go.mod" ] || ! grep -q "github.com/antimoji/antimoji" go.mod; then
    echo -e "${RED}Error: This script must be run from the antimoji project root${NC}"
    exit 1
fi

# Function to check file exists
check_file() {
    local file="$1"
    local description="$2"
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $description: $file"
        return 0
    else
        echo -e "${RED}✗${NC} $description: $file (missing)"
        return 1
    fi
}

# Function to check directory exists
check_dir() {
    local dir="$1"
    local description="$2"
    
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} $description: $dir"
        return 0
    else
        echo -e "${RED}✗${NC} $description: $dir (missing)"
        return 1
    fi
}

# Function to check command exists
check_command() {
    local cmd="$1"
    local description="$2"
    
    if command -v "$cmd" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $description: $(command -v $cmd)"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} $description: $cmd (not installed)"
        return 1
    fi
}

echo ""
echo -e "${BLUE}1. Core Files Check${NC}"
echo "-------------------"

files_ok=true
check_file "go.mod" "Go module file" || files_ok=false
check_file "Makefile" "Build configuration" || files_ok=false
check_file ".goreleaser.yaml" "Release configuration" || files_ok=false
check_file "Dockerfile" "Docker configuration" || files_ok=false

echo ""
echo -e "${BLUE}2. Workflow Files Check${NC}"
echo "-----------------------"

workflows_ok=true
check_file ".github/workflows/ci.yml" "CI workflow" || workflows_ok=false
check_file ".github/workflows/release.yml" "Release workflow" || workflows_ok=false
check_file ".github/workflows/build.yml" "Build workflow" || workflows_ok=false
check_file ".github/workflows/maintenance.yml" "Maintenance workflow" || workflows_ok=false
check_file ".github/workflows/nightly.yml" "Nightly workflow" || workflows_ok=false

echo ""
echo -e "${BLUE}3. Documentation Check${NC}"
echo "----------------------"

docs_ok=true
check_file "docs/ci-cd-setup.md" "CI/CD setup guide" || docs_ok=false
check_file "docs/release-process.md" "Release process guide" || docs_ok=false
check_file "docs/github-setup.md" "GitHub setup guide" || docs_ok=false
check_file "README.md" "Project README" || docs_ok=false
check_file "CHANGELOG.md" "Changelog" || docs_ok=false

echo ""
echo -e "${BLUE}4. Development Tools Check${NC}"
echo "----------------------------"

tools_ok=true
check_command "go" "Go compiler" || tools_ok=false
check_command "git" "Git version control" || tools_ok=false
check_command "make" "Make build tool" || tools_ok=false

# Optional tools
echo ""
echo -e "${BLUE}5. Optional Tools Check${NC}"
echo "-------------------------"

check_command "goreleaser" "GoReleaser (install with: go install github.com/goreleaser/goreleaser@latest)"
check_command "golangci-lint" "golangci-lint (install with: make install-tools)"
check_command "gosec" "gosec security scanner (install with: make install-tools)"
check_command "docker" "Docker (for container builds)"

echo ""
echo -e "${BLUE}6. Git Configuration Check${NC}"
echo "----------------------------"

git_ok=true

# Check if git remote is configured
if git remote get-url origin >/dev/null 2>&1; then
    remote_url=$(git remote get-url origin)
    echo -e "${GREEN}✓${NC} Git remote configured: $remote_url"
    
    # Check if remote is reachable
    if git ls-remote origin >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Remote repository accessible"
    else
        echo -e "${YELLOW}⚠${NC} Remote repository not accessible (may need authentication)"
        git_ok=false
    fi
else
    echo -e "${RED}✗${NC} Git remote not configured"
    echo -e "  ${YELLOW}Run: ./scripts/setup-github.sh${NC}"
    git_ok=false
fi

# Check git user configuration
if git config user.name >/dev/null 2>&1 && git config user.email >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Git user configured: $(git config user.name) <$(git config user.email)>"
else
    echo -e "${YELLOW}⚠${NC} Git user not fully configured"
    echo -e "  ${YELLOW}Run: git config --global user.name 'Your Name'${NC}"
    echo -e "  ${YELLOW}Run: git config --global user.email 'your.email@example.com'${NC}"
fi

echo ""
echo -e "${BLUE}7. Build Test${NC}"
echo "-------------"

build_ok=true
echo "Testing local build..."

if make build >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Local build successful"
    
    if [ -f "bin/antimoji" ]; then
        version_output=$(./bin/antimoji version 2>/dev/null || echo "version check failed")
        echo -e "${GREEN}✓${NC} Binary executable: $version_output"
    else
        echo -e "${RED}✗${NC} Binary not found after build"
        build_ok=false
    fi
else
    echo -e "${RED}✗${NC} Local build failed"
    echo -e "  ${YELLOW}Run: make build${NC} to see detailed error"
    build_ok=false
fi

echo ""
echo -e "${BLUE}8. Overall Status${NC}"
echo "-----------------"

overall_ok=true

if [ "$files_ok" = true ]; then
    echo -e "${GREEN}✓${NC} Core files: OK"
else
    echo -e "${RED}✗${NC} Core files: Missing files"
    overall_ok=false
fi

if [ "$workflows_ok" = true ]; then
    echo -e "${GREEN}✓${NC} GitHub workflows: OK"
else
    echo -e "${RED}✗${NC} GitHub workflows: Missing files"
    overall_ok=false
fi

if [ "$docs_ok" = true ]; then
    echo -e "${GREEN}✓${NC} Documentation: OK"
else
    echo -e "${RED}✗${NC} Documentation: Missing files"
    overall_ok=false
fi

if [ "$tools_ok" = true ]; then
    echo -e "${GREEN}✓${NC} Required tools: OK"
else
    echo -e "${RED}✗${NC} Required tools: Missing tools"
    overall_ok=false
fi

if [ "$git_ok" = true ]; then
    echo -e "${GREEN}✓${NC} Git configuration: OK"
else
    echo -e "${YELLOW}⚠${NC} Git configuration: Needs setup"
fi

if [ "$build_ok" = true ]; then
    echo -e "${GREEN}✓${NC} Build system: OK"
else
    echo -e "${RED}✗${NC} Build system: Issues found"
    overall_ok=false
fi

echo ""
if [ "$overall_ok" = true ] && [ "$git_ok" = true ]; then
    echo -e "${GREEN}🚀 CI/CD system is ready!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Push your code: git push origin main"
    echo "2. Watch CI workflow: https://github.com/YOUR_USERNAME/antimoji/actions"
    echo "3. Create first release: make release-prepare VERSION=v1.0.0"
    echo ""
elif [ "$overall_ok" = true ]; then
    echo -e "${YELLOW}⚠ CI/CD system is mostly ready, but git remote needs setup${NC}"
    echo ""
    echo "Run: ./scripts/setup-github.sh"
    echo ""
else
    echo -e "${RED}❌ CI/CD system needs fixes before use${NC}"
    echo ""
    echo "Issues to resolve:"
    [ "$files_ok" = false ] && echo "- Missing core files"
    [ "$workflows_ok" = false ] && echo "- Missing workflow files"
    [ "$docs_ok" = false ] && echo "- Missing documentation"
    [ "$tools_ok" = false ] && echo "- Missing required tools"
    [ "$build_ok" = false ] && echo "- Build system issues"
    echo ""
fi

echo "For detailed setup instructions, see:"
echo "- docs/github-setup.md"
echo "- docs/ci-cd-setup.md"
echo "- docs/release-process.md"
