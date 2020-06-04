#!/bin/bash

set -e

if [ -z "$FINDY_AGENT_ECR_URL" ]; then
  echo "ERROR: Define env variable FINDY_AGENT_ECR_URL"
  exit 1
fi

if [ -z "$FINDY_AGENT_ECR_REPOSITORY" ]; then
  echo "ERROR: Define env variable FINDY_AGENT_ECR_REPOSITORY"
  exit 1
fi

CURRENT_DIR=$(dirname "$BASH_SOURCE")

VERSION=$(cat $CURRENT_DIR/../VERSION)

echo "Releasing findy-agent version $VERSION"

docker rmi findy-agent || true
cd $CURRENT_DIR/..
make image

ECR_LOGIN=$(aws ecr get-login --no-include-email)
eval $ECR_LOGIN

docker tag findy-agent:latest $FINDY_AGENT_ECR_REPOSITORY:$VERSION
docker tag findy-agent:latest $FINDY_AGENT_ECR_REPOSITORY:latest
docker push $FINDY_AGENT_ECR_REPOSITORY

docker logout $FINDY_AGENT_ECR_URL
