#!/bin/bash

set -e

if [ -z "$1" ]; then
  echo "ERROR: Give new version name as first argument"
  exit 1
fi

VERSION_NBR=$(cat VERSION)
echo "Attempt to release version $VERSION_NBR"

BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [[ "$BRANCH" != "dev" ]]; then
  echo "ERROR: Checkout dev branch before tagging.";
  exit 1;
fi

if [ -z "$(git status --porcelain)" ]; then
  git pull origin dev

  VERSION=v$VERSION_NBR
  go mod tidy
  go test ./...

  set +e
  git commit -a -m "Releasing version $VERSION."
  set -e
  git tag -a $VERSION -m "Version $VERSION"
  git push origin dev --tags

  NEW_VERSION=$1
  echo $NEW_VERSION > VERSION
  git commit -a -m "Start dev for v$NEW_VERSION."
  git push origin dev
else 
  echo "ERROR: Working directory is not clean, commit or stash changes.";
fi


