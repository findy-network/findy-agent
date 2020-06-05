DEFAULT_APP_VERSION:=invalid
UPDATE_APP_VERSION?=invalid
AWS_ACCOUNT_ID=$(shell ../tools/aws-account-id.sh)
SECRETS_BUCKET=$(strip findy-agent-secrets-$(AWS_ACCOUNT_ID))
CONF_BUCKET=$(strip findy-agent-beanstalk-configuration-$(AWS_ACCOUNT_ID))

validate-ecr:
	aws cloudformation validate-template --template-body file://ecr.yaml

deploy-ecr:
	$(eval ECR_POLICY_JSON=$(shell cat ./ecr-lifecycle-policy.json))
	aws cloudformation deploy  \
		--stack-name findy-agent-ecr \
		--template-file ./ecr.yaml \
		--parameter-overrides EnvRepositoryLifecyclePolicyText='$(ECR_POLICY_JSON)'

delete-ecr:
	aws cloudformation delete-stack  \
		--stack-name findy-agent-ecr

validate-env:
	aws cloudformation validate-template \
		--template-body file://env.yaml

deploy-env:
	$(eval DEFAULT_APP_VERSION=$(shell ./default-app-version.sh))
	@echo "Deploying initial app version: $(DEFAULT_APP_VERSION)"
	aws cloudformation deploy  \
		--stack-name findy-agent-env \
		--template-file ./env.yaml \
		--capabilities CAPABILITY_NAMED_IAM \
		--parameter-overrides EnvBeanstalkApplicationVersionParameter=$(DEFAULT_APP_VERSION)

delete-env:
	aws cloudformation delete-stack  \
		--stack-name findy-agent-env

validate-init:
	aws cloudformation validate-template --template-body file://init-s3.yaml
	aws cloudformation validate-template --template-body file://init-env.yaml

deploy-init-s3:
	aws cloudformation deploy  \
		--stack-name findy-agent-init-s3 \
		--template-file ./init-s3.yaml

deploy-copy-s3-conf:
	sed 's/%BUCKET_NAME/$(SECRETS_BUCKET)/g' ./templates/s3.config.template > ./.ebextensions/s3.config.template
	sed 's/%BUCKET_URL/https:\/\/s3-$(AWS_DEFAULT_REGION).amazonaws.com\/$(SECRETS_BUCKET)/g' ./.ebextensions/s3.config.template > ./.ebextensions/s3.config
	rm ./.ebextensions/s3.config.template
	sed 's/%FINDY_AGENT_ECR_REPOSITORY/$(FINDY_AGENT_ECR_URL)\/findy-agent/g' ./templates/Dockerrun.aws.json.template > ./Dockerrun.aws.json
	zip -r Dockerrun.zip Dockerrun.aws.json .ebextensions
	aws s3 cp Dockerrun.zip s3://$(CONF_BUCKET)
	rm Dockerrun.zip

deploy-copy-s3-secrets:
	aws s3 sync .secrets s3://$(SECRETS_BUCKET)

deploy-init: deploy-init-s3 deploy-copy-s3-conf deploy-copy-s3-secrets
	aws cloudformation deploy  \
		--stack-name findy-agent-init-env \
		--template-file ./init-env.yaml

delete-init:
	aws cloudformation delete-stack  \
		--stack-name findy-agent-init-env
	-aws s3 rm s3://$(CONF_BUCKET) --recursive
	-aws s3 rm s3://$(SECRETS_BUCKET) --recursive
	aws cloudformation delete-stack  \
		--stack-name findy-agent-init-s3

update-env:
	aws cloudformation deploy  \
		--stack-name findy-agent-env \
		--template-file ./env.yaml \
		--capabilities CAPABILITY_NAMED_IAM \
		--parameter-overrides EnvBeanstalkApplicationVersionParameter=$(UPDATE_APP_VERSION)

deploy: deploy-init deploy-env

delete: delete-env delete-init

