#!/bin/bash
# Determine version based on git state:
# - If HEAD is tagged: use the tag (e.g., v1.0.0 -> 1.0.0)
# - If HEAD is not tagged but has previous tags: <latest-tag>-<short-hash> (e.g., 0.1.0-abc1234)
# - If no tags exist: commit hash only (e.g., abc1234)

set -euo pipefail

# Check if HEAD is exactly on a tag
TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")

if [ -n "$TAG" ]; then
    # Remove 'v' prefix if present
    echo "${TAG#v}"
    exit 0
fi

# HEAD is not on a tag, check if there are any previous tags
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
SHORT_HASH=$(git rev-parse --short HEAD)

if [ -n "$LATEST_TAG" ]; then
    # Remove 'v' prefix if present and append commit hash
    echo "${LATEST_TAG#v}-${SHORT_HASH}"
else
    # No tags exist, use commit hash only
    echo "${SHORT_HASH}"
fi
