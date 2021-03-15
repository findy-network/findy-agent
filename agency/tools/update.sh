#!/bin/bash

set -e

CURRENT_DIR=$(dirname "$BASH_SOURCE")
AWS_ACCOUNT_ID=$($CURRENT_DIR/aws-account-id.sh)

VERSION=$(cat $CURRENT_DIR/../../VERSION)

cd $CURRENT_DIR/../infra

echo $AWS_ACCOUNT_ID

make deploy-copy-s3-conf

aws elasticbeanstalk create-application-version \
  --application-name findy-agency \
  --version-label $VERSION \
  --source-bundle "S3Bucket=findy-agency-beanstalk-configuration-$AWS_ACCOUNT_ID,S3Key=Dockerrun.zip"

make update-env UPDATE_APP_VERSION=$VERSION
