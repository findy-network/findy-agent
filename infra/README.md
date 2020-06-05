# Findy-agent deployment

This folder contains scripts for deploying Findy-agent to AWS Elastic Beanstalk service.
*Note: this setup is intended only for development phase usage and does not consider fully security/scaling/recovery related requirements for production deployments.*

## Description

![Findy-Agent deployment in EB](../docs/infra-desc.png?raw=true 'Findy-Agent deployment in EB')

In this single container Docker environment findy-agent is run in AWS Elastic Beanstalk. The environment aims to work as a minimal setup for development purposes. There is no autoscaling or load balancer.

The findy-agent image is stored in AWS ECR from where the image is pulled during application updates. EBS volume is used as the persistent storage, all the needed configuration files are copied to the volume during the setup phase. Also the databases application needs are stored to the mounted volume.

Setup and update phase utilises also AWS Secrets Manager and S3 to provide the needed parameters and files to Elastic Beanstalk application.

The deployed application sends logs to AWS Cloudwatch. Logs can be easily monitored through AWS Console.

## Deployment

1. Install [AWS CLI](https://aws.amazon.com/cli/)

1. Create AWS user with access at least to following services:
   * CloudFormation
   * CloudWatch
   * CloudWatch Logs
   * EC2
   * EC2 Auto Scaling
   * Elastic Beanstalk
   * Elastic Container Registry
   * Elastic Container Service
   * IAM
   * S3
   * Secrets Manager

2. Define AWS environment variables
    ```
    export AWS_DEFAULT_REGION=xxx
    export AWS_ACCESS_KEY_ID=xxx
    export AWS_SECRET_ACCESS_KEY=xxx
    ```
1. Store findy-agent configuration to AWS Secrets Manager. Define following environment variables:
    ```
    # name for the steward wallet to be created
    export FINDY_DEV_LEDGER_STEWARD_WALLET_NAME=xxx

    # key for the steward wallet to be created
    export FINDY_DEV_LEDGER_STEWARD_WALLET_KEY=xxx

    # key for the wallet file where steward data is imported
    export FINDY_DEV_LEDGER_STEWARD_WALLET_IMPORTED_KEY=xxx

    # steward DID
    export FINDY_DEV_LEDGER_STEWARD_DID=xxx

    # salt string for findy-agent server
    export FINDY_AGENT_SALT=xxx

    # findy-agent server address as seen from the internet
    export FINDY_AGENT_HOST_ADDRESS=xxx
    ```

    Run script to store the variables to AWS:
    ```
    ./store-secrets.sh
    ```

1. Create ECR repository where to store the docker images:
    ```
    make deploy-ecr
    ```
    Note that you can use command `make delete-ecr`to remove the registry afterwards, but the possibly stored images need to be removed first.

1. Build and push the application image. Define following environment variables:
    ```
    # ECR URL for your AWS account
    export FINDY_AGENT_ECR_URL=xxx.amazonaws.com

    # Full ECR repository path (created in previous step)
    export FINDY_AGENT_ECR_REPOSITORY=$FINDY_AGENT_ECR_URL/findy-agent
    ```

    Build and push image to repository:
    ```
    ../tools/push.sh
    ```
1. Create folder `.secrets`. Place into folder following files:
    * **genesis_transactions** - ledger genesis file
    * **steward.exported** - exported steward wallet
    * **aps.p12** - Apple push notification certificate
1. Deploy the application with following command:
    ```
    make deploy
    ```
    You can remove the deployed CloudFormation stacks running: `make delete`.

## Update pipeline

![Findy-Agent pipeline](../docs/infra-pipeline.png?raw=true 'Findy-Agent pipeline')

New versions of findy-agent can be deployed with CI. [CI pipeline](../.circleci/config.yml) is launched when a new tag is created to the GitHub repository. CI builds the new image and pushes it to ECR. After that the new EB application configuration and version is updated.

*Note: updates to application secrets need to be done manually.*

Make sure the CI has following environment variables configured:
```
export AWS_DEFAULT_REGION=xxx
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
export FINDY_AGENT_ECR_URL=xxx
export FINDY_AGENT_ECR_REPOSITORY=xxx
```

### Tagging a release

Helper script for tagging a release can be found in tools-folder:
```
../tools/tag.sh 1.0
```
