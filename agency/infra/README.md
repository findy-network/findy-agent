# findy-agency deployment

This folder contains scripts for deploying findy-agency to AWS Elastic Beanstalk service.
_Note: this setup is intended only for development phase usage and does not consider fully security/scaling/recovery related requirements for production deployments._

## Description

![findy-agency deployment in EB](../../docs/infra-desc.png?raw=true "findy-agency deployment in EB")

In this single container Docker environment findy-agency is run in AWS Elastic Beanstalk. The environment aims to work as a minimal setup for development purposes. There is no autoscaling or load balancer.

The findy-agency image is stored in AWS ECR from where the image is pulled during application updates. EBS volume is used as the persistent storage, all the needed configuration files are copied to the volume during the setup phase. Also the application databases are stored to the mounted volume.

Setup and update phase utilises also AWS Secrets Manager and S3 to provide the needed parameters and files to Elastic Beanstalk application.

The deployed application sends logs to AWS Cloudwatch. Logs can be easily monitored through AWS Console.

## Deployment

1. Install [AWS CLI](https://aws.amazon.com/cli/)

1. Your AWS user needs access at least to following services:

   - CloudFormation
   - CloudWatch
   - CloudWatch Logs
   - EC2
   - EC2 Auto Scaling
   - Elastic Beanstalk
   - Elastic Container Registry
   - Elastic Container Service
   - IAM
   - S3
   - Secrets Manager

1. Define AWS environment variables

   ```bash
   export AWS_DEFAULT_REGION=xxx
   export AWS_ACCESS_KEY_ID=xxx
   export AWS_SECRET_ACCESS_KEY=xxx
   ```

1. Store findy-agency configuration to AWS Secrets Manager. Define following environment variables:

   ```bash
   # key for the steward wallet to be created
   export FINDY_AGENCY_STEWARD_WALLET_KEY=xxx

   # key for the wallet file where steward data is imported from
   export FINDY_AGENCY_STEWARD_WALLET_IMPORTED_KEY=xxx

   # steward DID
   export FINDY_AGENCY_STEWARD_DID=xxx

   # salt string for findy-agency server
   export FINDY_AGENCY_SALT=xxx

   # findy-agency server address as seen from the internet
   export FINDY_AGENCY_HOST_ADDRESS=xxx
   ```

   Run script to store the variables to AWS:

   ```bash
   ./store-secrets.sh
   ```

1. Create ECR repository where to store the docker images:

   ```bash
   make deploy-ecr
   ```

   Note that you can use command `make delete-ecr`to remove the registry afterwards, but the possibly stored images need to be removed first.

1. Build and push the application image. Define following environment variables:

   ```bash
   # ECR URL for your AWS account
   export FINDY_AGENCY_ECR_URL=xxx.amazonaws.com

   # Full ECR repository path (created in previous step)
   export FINDY_AGENCY_ECR_REPOSITORY=$FINDY_AGENCY_ECR_URL/findy-agency
   ```

   Build and push image to repository:

   ```bash
   ../tools/push.sh
   ```

1. Create folder `.secrets`. Place into folder following files:

   - **genesis_transactions** - ledger genesis file
   - **steward.exported** - exported steward wallet
   - **aps.p12** - Apple push notification certificate

1. Deploy the application with following command:

   ```bash
   make deploy
   ```

   You can remove the deployed CloudFormation stacks running: `make delete`.

## Update pipeline

![findy-agency pipeline](../../docs/infra-pipeline.png?raw=true "findy-agency pipeline")

New versions of findy-agency can be deployed e.g. with CI as described above. CI builds the new image and pushes it to ECR. After that the new EB application configuration and version is updated.

_Note: updates to application secrets need to be done manually._

Make sure the CI has following environment variables configured:

```bash
export AWS_DEFAULT_REGION=xxx
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
export FINDY_AGENCY_ECR_URL=xxx
export FINDY_AGENCY_ECR_REPOSITORY=xxx
```

Trigger push- and update-scripts for example whenever a tag is pushed to repository:

```bash
# Builds and pushes the new image
../tools/push.sh

# Updates the application version
../tools/update.sh
```

### Tagging a release

Helper script for tagging a release can be found in tools-folder:

```bash
../tools/tag.sh 1.0
```
