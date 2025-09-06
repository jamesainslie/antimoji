#!/bin/bash
# Update coverage badge in README from CI results

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Updating Coverage Badge${NC}"
echo "======================"

# Check if we're in the right directory
if [ ! -f "go.mod" ] || ! grep -q "github.com/antimoji/antimoji" go.mod; then
    echo -e "${RED}Error: This script must be run from the antimoji project root${NC}"
    exit 1
fi

# Run tests and generate coverage
echo "Running tests and generating coverage..."
go test -coverprofile=coverage.out -covermode=atomic ./...

# Calculate coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
echo -e "${BLUE}Current coverage: ${COVERAGE}%${NC}"

# Determine badge color based on coverage
if (( $(echo "$COVERAGE >= 90" | bc -l) )); then
    COLOR="brightgreen"
    STATUS="excellent"
elif (( $(echo "$COVERAGE >= 80" | bc -l) )); then
    COLOR="green"
    STATUS="good"
elif (( $(echo "$COVERAGE >= 70" | bc -l) )); then
    COLOR="yellow"
    STATUS="fair"
elif (( $(echo "$COVERAGE >= 60" | bc -l) )); then
    COLOR="orange"
    STATUS="poor"
else
    COLOR="red"
    STATUS="critical"
fi

echo -e "${BLUE}Badge color: ${COLOR} (${STATUS})${NC}"

# Create the badge URL
BADGE_URL="https://img.shields.io/badge/coverage-${COVERAGE}%25-${COLOR}?logo=go"

# Update README.md with the new badge
if [ -f "README.md" ]; then
    # Create a backup
    cp README.md README.md.backup

    # Update the coverage badge line
    sed -i.tmp "s|https://img.shields.io/github/commit-activity/m/jamesainslie/antimoji[^\"]*|${BADGE_URL}|g" README.md
    sed -i.tmp 's/alt="Commit Activity"/alt="Test Coverage"/g' README.md

    # Clean up temp file
    rm README.md.tmp

    echo -e "${GREEN}✓${NC} Updated coverage badge in README.md"
    echo -e "${BLUE}New badge URL:${NC} $BADGE_URL"
else
    echo -e "${RED}✗${NC} README.md not found"
    exit 1
fi

# Show the difference
echo ""
echo -e "${BLUE}Badge Update Summary:${NC}"
echo "Coverage: ${COVERAGE}%"
echo "Color: ${COLOR}"
echo "Status: ${STATUS}"
echo "URL: ${BADGE_URL}"

echo ""
echo -e "${GREEN}Coverage badge updated successfully!${NC}"
echo "Commit the changes with: git add README.md && git commit -m \"chore: update coverage badge to ${COVERAGE}%\""
