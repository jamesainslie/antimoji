#!/bin/bash
# Antimoji Pre-commit Hook
# Automatically cleans emojis from staged source files before commit

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Running antimoji pre-commit auto-clean...${NC}"

# Check if antimoji is built
if [ ! -f "bin/antimoji" ]; then
    echo -e "${YELLOW}⚠️ Antimoji binary not found. Building...${NC}"
    make build > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Failed to build antimoji${NC}"
        exit 1
    fi
fi

# Check if configuration exists
if [ ! -f ".antimoji.yaml" ]; then
    echo -e "${YELLOW}⚠️ No .antimoji.yaml found. Generating strict allowlist...${NC}"
    ./bin/antimoji generate --type=ci-lint --output=.antimoji.yaml . > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Failed to generate allowlist configuration${NC}"
        exit 1
    fi
fi

# Get list of staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

if [ -z "$STAGED_FILES" ]; then
    echo -e "${GREEN}✅ No staged files to check${NC}"
    exit 0
fi

echo -e "${BLUE}📁 Checking $(echo "$STAGED_FILES" | wc -l | tr -d ' ') staged files...${NC}"

# Create temporary file list for antimoji
TEMP_FILE_LIST=$(mktemp)
TEMP_RESULT_FILE=$(mktemp)
echo "$STAGED_FILES" > "$TEMP_FILE_LIST"

# Function to cleanup temp files
cleanup() {
    rm -f "$TEMP_FILE_LIST"
    rm -f "$TEMP_RESULT_FILE"
}
trap cleanup EXIT

# Auto-clean emojis from staged source files
echo -e "${BLUE}🧹 Auto-cleaning emojis from staged source files...${NC}"

CLEANED_FILES=0
TOTAL_EMOJIS_REMOVED=0

while IFS= read -r file; do
    if [ -f "$file" ]; then
        # Check if file should be processed (is it a source file?)
        case "$file" in
            *.go|*.js|*.ts|*.jsx|*.tsx|*.py|*.rb|*.java|*.c|*.cpp|*.h|*.hpp|*.rs|*.php|*.swift|*.kt|*.scala)
                # Skip test files and other excluded patterns
                case "$file" in
                    *_test.go|*/test/*|*/tests/*|*/testdata/*|*/fixtures/*|*/mocks/*|README.md|CHANGELOG.md|docs/*|.github/*|scripts/*|vendor/*|dist/*|bin/*|generate.go|config.go|allowlist.go|detector.go|root.go|detection.go)
                        continue
                        ;;
                    *)
                        echo -e "${BLUE}  Processing: $file${NC}"
                        
                        # First, check if file has emojis
                        SCAN_OUTPUT=$(./bin/antimoji scan --config=.antimoji.yaml --profile=pre-commit --ignore-allowlist --format=table "$file" 2>&1)
                        EMOJI_COUNT=$(echo "$SCAN_OUTPUT" | grep "Summary:" | awk '{print $2}')
                        
                        if [ -n "$EMOJI_COUNT" ] && [ "$EMOJI_COUNT" -gt 0 ]; then
                            echo -e "${YELLOW}    Found $EMOJI_COUNT emoji(s) - cleaning...${NC}"
                            
                            # Clean the file (with backup)
                            CLEAN_OUTPUT=$(./bin/antimoji clean --config=.antimoji.yaml --profile=pre-commit --in-place --backup --quiet "$file" 2>&1)
                            
                            if [ $? -eq 0 ]; then
                                echo -e "${GREEN}    ✅ Cleaned $file${NC}"
                                CLEANED_FILES=$((CLEANED_FILES + 1))
                                TOTAL_EMOJIS_REMOVED=$((TOTAL_EMOJIS_REMOVED + EMOJI_COUNT))
                                
                                # Re-stage the cleaned file
                                git add "$file"
                                echo -e "${CYAN}    📝 Re-staged cleaned file${NC}"
                            else
                                echo -e "${RED}    ❌ Failed to clean $file${NC}"
                                echo "$CLEAN_OUTPUT"
                                exit 1
                            fi
                        else
                            echo -e "${GREEN}    ✅ Already clean${NC}"
                        fi
                        ;;
                esac
                ;;
        esac
    fi
done < "$TEMP_FILE_LIST"

# Summary
if [ $CLEANED_FILES -gt 0 ]; then
    echo -e "${GREEN}🧹 Auto-clean complete!${NC}"
    echo -e "${CYAN}📊 Summary:${NC}"
    echo -e "  - Files cleaned: $CLEANED_FILES"
    echo -e "  - Emojis removed: $TOTAL_EMOJIS_REMOVED"
    echo -e "  - Backup files created: $CLEANED_FILES"
    echo -e "${YELLOW}💡 Backup files created with timestamp suffix${NC}"
    echo -e "${GREEN}✅ All staged files are now emoji-free and ready for commit!${NC}"
else
    echo -e "${GREEN}✅ No emojis found in staged source files${NC}"
    echo -e "${GREEN}✅ All files are clean - ready for commit!${NC}"
fi
