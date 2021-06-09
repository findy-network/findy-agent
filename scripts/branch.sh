#!/bin/bash

if [[ "$1" == "" ]]; then
	echo "Usage: ""0$" "<project_name>"
fi

{
pushd "$1" || exit 1
branch=$(git rev-parse --abbrev-ref HEAD)
popd || exit 1
} &> /dev/null

echo "$branch"
