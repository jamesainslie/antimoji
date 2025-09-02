#!/bin/bash
# Complete CI/CD Setup Script using GitHub CLI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Antimoji CI/CD Setup with GitHub CLI${NC}"
echo "====================================="

# Check if gh CLI is installed and authenticated
if ! command -v gh >/dev/null 2>&1; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
    echo "Install it from: https://cli.github.com/"
    exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
    echo -e "${RED}Error: GitHub CLI is not authenticated${NC}"
    echo "Run: gh auth login"
    exit 1
fi

echo -e "${GREEN}✓${NC} GitHub CLI is ready"

# Check if we're in the right directory
if [ ! -f "go.mod" ] || ! grep -q "github.com/antimoji/antimoji" go.mod; then
    echo -e "${RED}Error: This script must be run from the antimoji project root${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} In correct project directory"

# Check if repository exists or create it
echo ""
echo -e "${BLUE}1. Repository Setup${NC}"
echo "-------------------"

repo_name="antimoji"
if gh repo view "$repo_name" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Repository already exists: $(gh repo view --json url -q .url)"
else
    echo "Creating GitHub repository..."
    gh repo create "$repo_name" \
        --public \
        --description "High-performance emoji detection and removal CLI tool" \
        --add-readme=false
    echo -e "${GREEN}✓${NC} Repository created"
fi

# Configure git remote if needed
if ! git remote get-url origin >/dev/null 2>&1; then
    echo "Configuring git remote..."
    repo_url=$(gh repo view --json sshUrl -q .sshUrl 2>/dev/null || gh repo view --json url -q .url)
    git remote add origin "$repo_url"
    echo -e "${GREEN}✓${NC} Git remote configured: $repo_url"
else
    echo -e "${GREEN}✓${NC} Git remote already configured: $(git remote get-url origin)"
fi

# Set default repository for gh commands
gh repo set-default "$(gh repo view --json nameWithOwner -q .nameWithOwner)"

echo ""
echo -e "${BLUE}2. Repository Configuration${NC}"
echo "---------------------------"

# Configure repository settings
echo "Configuring repository settings..."
gh repo edit \
    --enable-issues \
    --enable-projects \
    --enable-wiki \
    --enable-discussions \
    --enable-auto-merge \
    --enable-secret-scanning \
    --delete-branch-on-merge \
    --add-topic cli \
    --add-topic emoji \
    --add-topic go \
    --add-topic performance \
    --add-topic text-processing

echo -e "${GREEN}✓${NC} Repository settings configured"

echo ""
echo -e "${BLUE}3. Secrets Configuration${NC}"
echo "------------------------"

# Check for optional secrets
echo "Checking for optional secrets..."

# List current secrets
echo "Current secrets:"
if gh secret list --json name | grep -q "name"; then
    gh secret list --json name -q '.[].name' | while read secret; do
        echo -e "${GREEN}✓${NC} $secret"
    done
else
    echo "No secrets configured (this is fine - GITHUB_TOKEN is automatic)"
fi

echo ""
echo "Optional secrets you can configure:"
echo "- CODECOV_TOKEN: For code coverage reporting (get from codecov.io)"
echo "- HOMEBREW_TOKEN: For automated Homebrew formula updates"
echo ""
echo "To add a secret: gh secret set SECRET_NAME"

echo ""
echo -e "${BLUE}4. Push Code and Test CI/CD${NC}"
echo "------------------------------"

# Check if we need to push
if git diff origin/main --quiet >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Code is up to date with remote"
else
    echo "Pushing code to trigger CI/CD..."
    
    # Try to push, handle conflicts gracefully
    if git push origin main; then
        echo -e "${GREEN}✓${NC} Code pushed successfully"
    else
        echo -e "${YELLOW}⚠${NC} Push failed - may need to pull and merge first"
        echo "Run: git pull origin main --no-rebase"
        echo "Then: git push origin main"
    fi
fi

echo ""
echo -e "${BLUE}5. Monitor Workflows${NC}"
echo "-------------------"

echo "Checking workflow status..."
if gh run list --limit 1 --json status,conclusion,displayTitle | grep -q "in_progress"; then
    echo -e "${YELLOW}⚠${NC} Workflows are currently running"
    echo "Monitor with: gh run watch"
else
    echo "Latest workflow runs:"
    gh run list --limit 3
fi

echo ""
echo -e "${BLUE}6. Setup Complete${NC}"
echo "-----------------"

echo -e "${GREEN}✅ CI/CD system is now configured and active!${NC}"
echo ""
echo "What happens next:"
echo "1. Every push to main/develop triggers CI workflow"
echo "2. Every PR triggers CI with benchmarks"
echo "3. Pushing version tags (v1.0.0) triggers releases"
echo "4. Weekly maintenance checks run automatically"
echo "5. Nightly comprehensive testing runs daily"
echo ""
echo "Useful commands:"
echo "• Watch workflows: gh run watch"
echo "• View workflow logs: gh run view <run-id>"
echo "• Create release: make release-prepare VERSION=v1.0.0 && git push origin v1.0.0"
echo "• Check CI status: ./scripts/check-ci-status.sh"
echo ""
echo "Repository: $(gh repo view --json url -q .url)"
echo "Actions: $(gh repo view --json url -q .url)/actions"
echo ""
echo "Documentation:"
echo "- docs/ci-cd-setup.md"
echo "- docs/release-process.md"
echo "- docs/github-setup.md"
