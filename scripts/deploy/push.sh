#!/bin/bash

set -e

AWS_CMD="docker run --rm -it \
    -e AWS_DEFAULT_REGION=$AWS_DEFAULT_REGION \
    -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    amazon/aws-cli"

if [ -z "$ECR_IMAGE_NAME" ]; then
  echo "ERROR: Define env variable ECR_IMAGE_NAME"
  exit 1
fi

if [ -z "$ECR_ROOT_URL" ]; then
  echo "ERROR: Define env variable ECR_ROOT_URL"
  exit 1
fi

FULL_NAME="$ECR_ROOT_URL/$ECR_IMAGE_NAME"
CURRENT_DIR=$(dirname "$BASH_SOURCE")

VERSION=$(cat $CURRENT_DIR/../../VERSION)

echo "Checking if $VERSION is already built..."

set +e
HAS_IMAGE_VERSION=$($AWS_CMD ecr list-images --repository-name $ECR_IMAGE_NAME --filter '{"tagStatus": "TAGGED"}' | grep $VERSION)
set -e

if [ -z "$HAS_IMAGE_VERSION" ]; then
  echo "Image $VERSION not found in registry, start building.";
else
  echo "WARNING: Image $VERSION already built, skipping build!";
  exit 0
fi

echo "Releasing findy-agent version $VERSION"

cd $CURRENT_DIR/../..
make agency

$AWS_CMD ecr get-login-password \
    --region $AWS_DEFAULT_REGION \
| docker login \
    --username AWS \
    --password-stdin $ECR_ROOT_URL

docker tag findy-agency:latest $FULL_NAME:$VERSION
docker tag findy-agency:latest $FULL_NAME:latest
docker push $FULL_NAME

docker logout $ECR_ROOT_URL
