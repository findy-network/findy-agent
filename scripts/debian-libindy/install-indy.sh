#!/bin/bash

INDY_LIB_VERSION="1.16.0"
UBUNTU_VERSION="bionic"

sudo apt-get update && \
    sudo apt-get install -y software-properties-common apt-transport-https && \
    sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 68DB5E88 && \
    sudo add-apt-repository "deb https://repo.sovrin.org/sdk/deb $UBUNTU_VERSION stable" && \
    sudo add-apt-repository "deb https://repo.sovrin.org/sdk/deb xenial stable" && \
    sudo apt-get update

sudo apt-get install -y libindy-dev="$INDY_LIB_VERSION-xenial" \
    libindy="$INDY_LIB_VERSION-$UBUNTU_VERSION"
