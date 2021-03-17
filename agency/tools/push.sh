#!/bin/bash

set -e

if [ -z "$FINDY_AGENCY_ECR_URL" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_ECR_URL"
  exit 1
fi

if [ -z "$FINDY_AGENCY_ECR_REPOSITORY" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_ECR_REPOSITORY"
  exit 1
fi

CURRENT_DIR=$(dirname "$BASH_SOURCE")

VERSION=$(cat $CURRENT_DIR/../../VERSION)

echo "Releasing findy-agency version $VERSION"

docker rmi findy-agency || true
cd $CURRENT_DIR/../..
make agency

aws ecr get-login-password \
    --region $AWS_DEFAULT_REGION \
| docker login \
    --username AWS \
    --password-stdin $FINDY_AGENCY_ECR_URL

docker tag findy-agency:latest $FINDY_AGENCY_ECR_REPOSITORY:$VERSION
docker tag findy-agency:latest $FINDY_AGENCY_ECR_REPOSITORY:latest
docker push $FINDY_AGENCY_ECR_REPOSITORY

docker logout $FINDY_AGENCY_ECR_URL
