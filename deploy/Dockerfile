FROM ubuntu:18.04

VOLUME /root/.indy_client

RUN apt-get update && \
    apt-get install -y software-properties-common apt-transport-https && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 68DB5E88 && \
    add-apt-repository "deb https://repo.sovrin.org/sdk/deb xenial stable" && \
    apt-get install -y wget multiarch-support && \
    apt-get update && \
    wget "http://security-cdn.debian.org/debian-security/pool/updates/main/o/openssl/libssl1.0.0_1.0.1t-1+deb8u11_amd64.deb" && \
    dpkg -i libssl1.0.0_1.0.1t-1+deb8u11_amd64.deb && \
    apt-get install -y libssl-dev libindy=1.8.3 indy-cli=1.8.3

COPY ./bin/findy-agent /root/findy-agent

EXPOSE 8080

CMD /root/findy-agent