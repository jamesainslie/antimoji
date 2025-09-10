#!/bin/bash
set -e

echo "Committing PR #9 changes..."

git commit -m "Complete CLI layer migration to dependency injection

- Refactor generate command to use dependency injection
- Refactor setup-lint command to use dependency injection
- Add complete clean command implementation with DI
- Update application.go to use all new command handlers
- All CLI commands now use consistent DI pattern
- Maintain backward compatibility for all flags
- All tests passing, linting clean"

echo "Commit completed successfully!"
