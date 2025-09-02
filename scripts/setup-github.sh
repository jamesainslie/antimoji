#!/bin/bash
# GitHub Repository Setup Script for Antimoji

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Antimoji GitHub Repository Setup${NC}"
echo "================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ] || ! grep -q "github.com/antimoji/antimoji" go.mod; then
    echo -e "${RED}Error: This script must be run from the antimoji project root${NC}"
    exit 1
fi

# Check if git is initialized
if [ ! -d ".git" ]; then
    echo -e "${RED}Error: Git repository not initialized${NC}"
    exit 1
fi

# Function to prompt for GitHub username
get_github_username() {
    while true; do
        read -p "Enter your GitHub username: " username
        if [ -n "$username" ]; then
            break
        fi
        echo "Please enter a valid username."
    done
    echo "$username"
}

# Function to choose remote type
choose_remote_type() {
    echo ""
    echo "Choose remote type:"
    echo "1) HTTPS (recommended for most users)"
    echo "2) SSH (if you have SSH keys configured)"
    read -p "Enter choice (1 or 2): " choice
    
    case $choice in
        1) echo "https" ;;
        2) echo "ssh" ;;
        *) echo "https" ;;
    esac
}

# Check if remote already exists
if git remote get-url origin >/dev/null 2>&1; then
    current_remote=$(git remote get-url origin)
    echo -e "${YELLOW}Git remote 'origin' already configured:${NC} $current_remote"
    read -p "Do you want to update it? (y/N): " update_remote
    
    if [[ $update_remote =~ ^[Yy]$ ]]; then
        git remote remove origin
        echo "Removed existing remote"
    else
        echo "Keeping existing remote configuration"
        exit 0
    fi
fi

# Get GitHub username
echo ""
username=$(get_github_username)

# Choose remote type
remote_type=$(choose_remote_type)

# Configure remote
echo ""
echo -e "${BLUE}Configuring git remote...${NC}"

if [ "$remote_type" = "ssh" ]; then
    remote_url="git@github.com:$username/antimoji.git"
else
    remote_url="https://github.com/$username/antimoji.git"
fi

git remote add origin "$remote_url"
echo -e "${GREEN}Added remote:${NC} $remote_url"

# Test connectivity
echo ""
echo -e "${BLUE}Testing remote connectivity...${NC}"
if git ls-remote origin >/dev/null 2>&1; then
    echo -e "${GREEN}Remote connectivity successful${NC}"
else
    echo -e "${YELLOW}Warning: Cannot connect to remote repository${NC}"
    echo "Make sure the repository exists and you have access to it"
    echo "Repository should be at: https://github.com/$username/antimoji"
fi

# Offer to push
echo ""
read -p "Do you want to push your code now? (Y/n): " push_now

if [[ ! $push_now =~ ^[Nn]$ ]]; then
    echo ""
    echo -e "${BLUE}Pushing to GitHub...${NC}"
    
    if git push -u origin main; then
        echo -e "${GREEN}Successfully pushed to GitHub${NC}"
        echo ""
        echo -e "${GREEN}Setup complete! Your CI/CD system is now active.${NC}"
        echo ""
        echo "Next steps:"
        echo "1. Go to https://github.com/$username/antimoji"
        echo "2. Check the Actions tab to see your CI workflow running"
        echo "3. Review docs/ci-cd-setup.md for configuration options"
        echo "4. Review docs/release-process.md for release instructions"
        echo ""
        echo "To create your first release:"
        echo "  make release-prepare VERSION=v1.0.0"
        echo "  git push origin v1.0.0"
    else
        echo -e "${RED}Failed to push to GitHub${NC}"
        echo "Please check your repository permissions and try again"
    fi
else
    echo ""
    echo -e "${GREEN}Git remote configured successfully${NC}"
    echo "To push later: git push -u origin main"
fi

echo ""
echo -e "${BLUE}Repository URL:${NC} https://github.com/$username/antimoji"
echo -e "${BLUE}Documentation:${NC} See docs/ directory for detailed guides"
