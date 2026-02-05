#!/bin/bash
# Bump version script

set -e

VERSION_FILE="VERSION"
CURRENT=$(cat $VERSION_FILE)

IFS='.' read -ra PARTS <<< "$CURRENT"
MAJOR=${PARTS[0]}
MINOR=${PARTS[1]}
PATCH=${PARTS[2]}

case "$1" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo "Usage: $0 {major|minor|patch}"
        exit 1
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"
echo "$NEW_VERSION" > $VERSION_FILE
echo "Version bumped: $CURRENT → $NEW_VERSION"
