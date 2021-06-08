#!/bin/bash

INDY_LIB_VERSION="1.16.0"
UBUNTU_VERSION="bionic"

apt-get update && \
    apt-get install -y software-properties-common apt-transport-https && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 68DB5E88 && \
    add-apt-repository "deb https://repo.sovrin.org/sdk/deb $UBUNTU_VERSION stable" && \
    add-apt-repository "deb https://repo.sovrin.org/sdk/deb xenial stable" && \
    apt-get update

apt-get install -y libindy-dev="$INDY_LIB_VERSION-xenial" \
    libindy="$INDY_LIB_VERSION-$UBUNTU_VERSION"
