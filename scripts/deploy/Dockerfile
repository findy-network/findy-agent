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

WORKDIR /work

COPY go.* ./
RUN go mod download

COPY . ./

RUN make install

FROM ghcr.io/findy-network/findy-base:indy-1.16.ubuntu-18.04

LABEL org.opencontainers.image.source https://github.com/findy-network/findy-agent

COPY --from=0 /go/bin/findy-agent /findy-agent
ADD ./scripts/deploy/import-wallet.sh .

RUN echo "{}" > /root/findy.json

EXPOSE 8080
EXPOSE 50051

# override when running container:

# debug levels 3 - 5 - 10 from less to more verbose
ENV FCLI_LOGGING -logtostderr=true -v=3

ENV FCLI_POOL_GENESIS_TXN_FILE /genesis_transactions
ENV FCLI_POOL_NAME findy

ENV FCLI_IMPORT_WALLET_FILE_KEY 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY
ENV FCLI_IMPORT_WALLET_KEY 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY
ENV FCLI_IMPORT_WALLET_NAME steward
ENV FCLI_IMPORT_WALLET_FILE /steward.exported

ENV FCLI_AGENCY_HOST_ADDRESS localhost
ENV FCLI_AGENCY_HOST_PORT 8080
ENV FCLI_AGENCY_SERVER_PORT 8080
ENV FCLI_AGENCY_POOL_NAME ${FCLI_POOL_NAME}
ENV FCLI_AGENCY_PSM_DATABASE_FILE /root/findy.bolt
ENV FCLI_AGENCY_REGISTER_FILE /root/findy.json
ENV FCLI_AGENCY_STEWARD_WALLET_NAME ${FCLI_IMPORT_WALLET_NAME}
ENV FCLI_AGENCY_STEWARD_WALLET_KEY ${FCLI_IMPORT_WALLET_KEY}
ENV FCLI_AGENCY_STEWARD_DID Th7MpTaRZVRYnPiabds81Y
ENV FCLI_AGENCY_SALT "this is only example"
ENV FCLI_AGENCY_GRPC_TLS false

ENV FCLI_AGENCY_APNS_P12_FILE ""

RUN echo '[[ ! -z "$STARTUP_FILE_STORAGE_S3" ]] && /s3-copy $STARTUP_FILE_STORAGE_S3 agent /' > /start.sh && \
    echo '[[ ! -z "$STARTUP_FILE_STORAGE_S3" ]] && /s3-copy $STARTUP_FILE_STORAGE_S3 grpc /grpc' >> /start.sh && \
    echo './import-wallet.sh' >> /start.sh && \
    echo '/findy-agent agency start --grpc=true' >> /start.sh && chmod a+x /start.sh

ENTRYPOINT ["/bin/bash", "-c", "/start.sh"]
