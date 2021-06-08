# Container image that runs your code
FROM golang:1.16-buster

ENV INDY_LIB_VERSION="1.16.0"
ENV UBUNTU_VERSION="bionic"

RUN apt-get update && \
    apt-get install -y software-properties-common apt-transport-https && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 68DB5E88 && \
    add-apt-repository "deb https://repo.sovrin.org/sdk/deb $UBUNTU_VERSION stable" && \
    add-apt-repository "deb https://repo.sovrin.org/sdk/deb xenial stable" && \
    apt-get update

RUN apt-get install -y libindy-dev="$INDY_LIB_VERSION-xenial" \
    libindy="$INDY_LIB_VERSION-$UBUNTU_VERSION"

# Copies your code file from your action repository to the filesystem path `/` of the container
COPY ./findy-agent/action/e2e/entrypoint.sh /entrypoint.sh
COPY ./findy-agent /findy-agent

# Code file to execute when the docker container starts up (`entrypoint.sh`)
ENTRYPOINT ["/entrypoint.sh"]
