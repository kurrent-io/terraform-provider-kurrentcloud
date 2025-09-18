#!/bin/bash

# Script to generate documentation for kurrentcloud resources only
# Keeps all resources registered for backwards compatibility but cleans up
# eventstorecloud documentation after generation

set -e

echo "Generating documentation for kurrentcloud resources only..."
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --ignore-deprecated=true

echo "Cleaning up eventstorecloud templates and documentation..."

# Remove any auto-generated eventstorecloud template files
find templates/ -name "*eventstorecloud*" -type f -delete 2>/dev/null || true

# Remove any auto-generated eventstorecloud documentation files
find docs/ -name "*eventstorecloud*" -type f -delete 2>/dev/null || true

# Also clean up any eventstorecloud files in current directory
find . -maxdepth 1 -name "*eventstorecloud*" -type f -delete 2>/dev/null || true

echo "Documentation generation completed successfully!"
echo "Only kurrentcloud documentation is available."