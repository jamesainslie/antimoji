#!/bin/bash
# Badge Status Checker for Antimoji

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

REPO="jamesainslie/antimoji"

echo -e "${BLUE}Antimoji Badge Status Check${NC}"
echo "=========================="
echo "Repository: $REPO"
echo ""

# Function to check URL status
check_url() {
    local url="$1"
    local description="$2"
    
    if curl -s --head "$url" | head -n 1 | grep -q "200 OK"; then
        echo -e "${GREEN}✓${NC} $description: Working"
        return 0
    else
        echo -e "${RED}✗${NC} $description: Not working"
        echo "  URL: $url"
        return 1
    fi
}

echo -e "${BLUE}1. CI/CD Workflow Badges${NC}"
echo "------------------------"

# CI Badge
check_url "https://github.com/$REPO/workflows/CI/badge.svg" "CI Workflow Badge"

# Build Badge  
check_url "https://github.com/$REPO/workflows/Build/badge.svg" "Build Workflow Badge"

# Nightly Build Badge
check_url "https://github.com/$REPO/workflows/Nightly%20Build/badge.svg" "Nightly Build Badge"

echo ""
echo -e "${BLUE}2. Repository Information Badges${NC}"
echo "-------------------------------"

# Go Version Badge
check_url "https://img.shields.io/github/go-mod/go-version/$REPO?color=00ADD8&logo=go" "Go Version Badge"

# Latest Release Badge
check_url "https://img.shields.io/github/v/release/$REPO?color=blue&logo=github" "Latest Release Badge"

# Downloads Badge
check_url "https://img.shields.io/github/downloads/$REPO/total?color=green&logo=github" "Total Downloads Badge"

# License Badge
check_url "https://img.shields.io/github/license/$REPO?color=blue" "License Badge"

echo ""
echo -e "${BLUE}3. Quality and Documentation Badges${NC}"
echo "----------------------------------"

# Coverage Badge (may not work until Codecov is set up)
echo -e "${YELLOW}⚠${NC} Coverage Badge: https://img.shields.io/codecov/c/github/$REPO?logo=codecov"
echo "  Note: Requires CODECOV_TOKEN secret to be configured"

# Go Report Card Badge
check_url "https://goreportcard.com/badge/github.com/$REPO" "Go Report Card Badge"

# Go Reference Badge
check_url "https://pkg.go.dev/badge/github.com/$REPO.svg" "Go Reference Badge"

echo ""
echo -e "${BLUE}4. Workflow Status Check${NC}"
echo "------------------------"

# Check recent workflow runs
echo "Recent workflow runs:"
if command -v gh >/dev/null 2>&1; then
    gh run list --limit 3 --json status,conclusion,displayTitle,createdAt | \
    jq -r '.[] | "  \(.displayTitle): \(.status) (\(.conclusion // "running"))"'
else
    echo "  Install 'gh' CLI to see workflow status"
fi

echo ""
echo -e "${BLUE}5. Badge URLs for Reference${NC}"
echo "-----------------------------"

echo "CI Badges:"
echo "  CI: https://github.com/$REPO/workflows/CI/badge.svg"
echo "  Build: https://github.com/$REPO/workflows/Build/badge.svg"
echo "  Nightly: https://github.com/$REPO/workflows/Nightly%20Build/badge.svg"
echo ""

echo "Repository Badges:"
echo "  Go Version: https://img.shields.io/github/go-mod/go-version/$REPO?color=00ADD8&logo=go"
echo "  Latest Release: https://img.shields.io/github/v/release/$REPO?color=blue&logo=github"
echo "  Downloads: https://img.shields.io/github/downloads/$REPO/total?color=green&logo=github"
echo "  License: https://img.shields.io/github/license/$REPO?color=blue"
echo ""

echo "Quality Badges:"
echo "  Coverage: https://img.shields.io/codecov/c/github/$REPO?logo=codecov"
echo "  Go Report: https://goreportcard.com/badge/github.com/$REPO"
echo "  Go Reference: https://pkg.go.dev/badge/github.com/$REPO.svg"
echo ""

echo -e "${BLUE}6. Badge Setup Instructions${NC}"
echo "----------------------------"

echo "To set up Codecov coverage badge:"
echo "1. Go to https://codecov.io/"
echo "2. Sign in with GitHub"
echo "3. Add your repository: $REPO"
echo "4. Get the upload token"
echo "5. Add as GitHub secret: gh secret set CODECOV_TOKEN"
echo ""

echo "To improve Go Report Card score:"
echo "1. Visit: https://goreportcard.com/report/github.com/$REPO"
echo "2. Review recommendations"
echo "3. Fix any issues identified"
echo ""

echo "Badge markdown for README:"
echo '```markdown'
echo "[![CI Status](https://github.com/$REPO/workflows/CI/badge.svg)](https://github.com/$REPO/actions/workflows/ci.yml)"
echo "[![Build Status](https://github.com/$REPO/workflows/Build/badge.svg)](https://github.com/$REPO/actions/workflows/build.yml)"
echo "[![Go Version](https://img.shields.io/github/go-mod/go-version/$REPO?color=00ADD8&logo=go)](https://golang.org)"
echo "[![Latest Release](https://img.shields.io/github/v/release/$REPO?color=blue&logo=github)](https://github.com/$REPO/releases)"
echo "[![License](https://img.shields.io/github/license/$REPO?color=blue)](LICENSE)"
echo "[![Go Report Card](https://goreportcard.com/badge/github.com/$REPO)](https://goreportcard.com/report/github.com/$REPO)"
echo '```'
