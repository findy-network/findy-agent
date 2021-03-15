# Container image that runs your code
FROM optechlab/indy-golang:1.16.0

# Copies your code file from your action repository to the filesystem path `/` of the container
COPY ./findy-agent/action/e2e/entrypoint.sh /entrypoint.sh
COPY ./findy-agent /findy-agent

# Code file to execute when the docker container starts up (`entrypoint.sh`)
ENTRYPOINT ["/entrypoint.sh"]
