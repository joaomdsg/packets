#!/bin/sh
set -e

# Configure git to use gh as credential helper
gh auth setup-git

# Run the main application
exec agntpr "$@"
