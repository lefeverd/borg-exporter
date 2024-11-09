#!/bin/bash
##
## This script can be executed to release the application.
## It will fetch the latest tag from the repository, bump it, create the new tag and push it,
## which will trigger the release pipeline.
##

DRY_RUN=false

# Check for at least one argument for VERSION_PART
if [[ -z "$1" ]]; then
  echo "Error: Missing version part argument (major, minor, or patch)."
  echo "Usage: $0 <major|minor|patch> [--dry-run]"
  exit 1
fi

# Parse arguments for version part and dry run option
for arg in "$@"; do
  case $arg in
    major|minor|patch)
      VERSION_PART=$arg
      shift
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    *)
      echo "Usage: $0 [major|minor|patch] [--dry-run]"
      exit 1
      ;;
  esac
done

# Fetch the latest tags from the remote
echo "Fetching latest tags..."
git fetch --tags

# Get the latest tag in the format vx.y.z
LATEST_TAG=$(git describe --tags $(git rev-list --tags --max-count=1) 2>/dev/null)

# If no tags are found, start with v0.0.0
LATEST_TAG=${LATEST_TAG:-v0.0.0}

# Extract major, minor, and patch versions
MAJOR=$(echo "$LATEST_TAG" | sed 's/^v\([0-9]*\)\..*/\1/')
MINOR=$(echo "$LATEST_TAG" | sed 's/^v[0-9]*\.\([0-9]*\)\..*/\1/')
PATCH=$(echo "$LATEST_TAG" | sed 's/^v[0-9]*\.[0-9]*\.\([0-9]*\)/\1/')

# Increment the appropriate version part
if [ "$VERSION_PART" == "major" ]; then
  NEW_VERSION="v$((MAJOR + 1)).0.0"
elif [ "$VERSION_PART" == "minor" ]; then
  NEW_VERSION="v$MAJOR.$((MINOR + 1)).0"
else
  NEW_VERSION="v$MAJOR.$MINOR.$((PATCH + 1))"
fi

echo "Latest tag: $LATEST_TAG"
echo "New version: $NEW_VERSION"

if [ "$DRY_RUN" = true ]; then
  echo "Dry run: Tag $NEW_VERSION will not be created or pushed."
else
  # Create and push the new tag
  git tag "$NEW_VERSION"
  git push origin "$NEW_VERSION"
  echo "Tag $NEW_VERSION has been created and pushed."
fi
