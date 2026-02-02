#!/bin/bash
set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 v1.0.0"
  exit 1
fi

VERSION=$1
DATE=$(date +%Y-%m-%d)
CHANGELOG_FILE="CHANGELOG.md"

if [ ! -f "$CHANGELOG_FILE" ]; then
  echo "Error: $CHANGELOG_FILE not found!"
  exit 1
fi

# Check if the version already exists
if grep -q "\[$VERSION\]" "$CHANGELOG_FILE"; then
  echo "Error: Version $VERSION already exists in $CHANGELOG_FILE"
  exit 1
fi

# Use awk to insert the new version section
TEMP_FILE=$(mktemp)

awk -v ver="$VERSION" -v date="$DATE" '
  /^## \[Unreleased\]$/ {
    print "## [Unreleased]"
    print ""
    print "## [" ver "] - " date
    next
  }
  { print }
' "$CHANGELOG_FILE" > "$TEMP_FILE"

mv "$TEMP_FILE" "$CHANGELOG_FILE"

echo "Successfully updated $CHANGELOG_FILE for release $VERSION"
