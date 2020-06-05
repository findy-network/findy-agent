#!/bin/bash

if [ -z "$1" ]; then
  echo "ERROR: Give version number as parameter e.g. 0.10";
  exit 1;
fi

VERSION_NBR=$1
echo "Attempt to release version $VERSION_NBR"

BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [[ "$BRANCH" != "master" ]]; then
  echo "ERROR: Checkout master branch before tagging.";
  exit 1;
fi

if [ -z "$(git status --porcelain)" ]; then
  VERSION=v$VERSION_NBR

  echo $VERSION_NBR > VERSION

  git commit -a -m "Releasing version $VERSION."
  git tag -a $VERSION -m "Version $VERSION"
  git push origin master --tags
else 
  echo "ERROR: Working directory is not clean, commit or stash changes.";
fi

