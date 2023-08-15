FROM ghcr.io/findy-network/findy-base:indy-1.16.ubuntu-22.04 AS indy-base

FROM golang:1.21-bookworm AS agent-builder

ARG GOBUILD_ARGS=""

ENV INDY_LIB_VERSION="1.16.0"

# install indy deps and files from base
RUN apt-get update && apt-get install -y libsodium23 libzmq5

COPY --from=indy-base /usr/include/indy /usr/include/indy
COPY --from=indy-base /usr/lib/libindy.a /usr/lib/libindy.a
COPY --from=indy-base /usr/lib/libindy.so /usr/lib/libindy.so

# manually copy libssl1.1 as it is no longer supported for newer ubuntu
COPY --from=indy-base /usr/lib/engines-1.1 /usr/lib/engines-1.1
COPY --from=indy-base /usr/lib/libcrypto.so.1.1 /usr/lib/libcrypto.so.1.1
COPY --from=indy-base /usr/lib/libssl.so.1.1 /usr/lib/libssl.so.1.1

WORKDIR /work

COPY go.* ./
RUN go mod download

COPY . ./

RUN make install GOBUILD_ARGS=${GOBUILD_ARGS}

FROM ghcr.io/findy-network/findy-base:indy-1.16.ubuntu-22.04

LABEL org.opencontainers.image.source https://github.com/findy-network/findy-agent

# TODO: remove (healthcheck utility)
RUN apt-get update && apt-get install -y curl

COPY --from=agent-builder /go/bin/findy-agent /findy-agent
ADD ./scripts/deploy/import-wallet.sh .
ADD ./scripts/deploy/healthcheck.sh .

EXPOSE 8080
EXPOSE 50051

# used when running instrumented binary
ENV GOCOVERDIR /coverage

# override when running container:

# debug levels 3 - 5 - 10 from less to more verbose
ENV FCLI_LOGGING "-logtostderr=true -v=3"

ENV FCLI_POOL_GENESIS_TXN_FILE "/genesis_transactions"
ENV FCLI_POOL_NAME "findy"

ENV FCLI_IMPORT_WALLET_FILE_KEY "9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY"
ENV FCLI_IMPORT_WALLET_KEY "9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY"
ENV FCLI_IMPORT_WALLET_NAME "steward"
ENV FCLI_IMPORT_WALLET_FILE "/steward.exported"

ENV FCLI_AGENCY_HOST_ADDRESS "localhost"
ENV FCLI_AGENCY_HOST_PORT "8080"
ENV FCLI_AGENCY_SERVER_PORT "8080"
ENV FCLI_AGENCY_POOL_NAME "FINDY_LEDGER,${FCLI_POOL_NAME}"
ENV FCLI_AGENCY_PSM_DATABASE_FILE "/root/findy.bolt"
ENV FCLI_AGENCY_REGISTER_FILE "/root/findy.json"
ENV FCLI_AGENCY_STEWARD_WALLET_NAME "${FCLI_IMPORT_WALLET_NAME}"
ENV FCLI_AGENCY_STEWARD_WALLET_KEY "${FCLI_IMPORT_WALLET_KEY}"
ENV FCLI_AGENCY_STEWARD_DID "Th7MpTaRZVRYnPiabds81Y"
ENV FCLI_AGENCY_GRPC_TLS "false"
ENV FCLI_AGENCY_REQUEST_TIMEOUT "3m"
ENV FCLI_AGENCY_WALLET_POOL "400"

ENV FCLI_STEWARD_WALLET_NAME "${FCLI_IMPORT_WALLET_NAME}"
ENV FCLI_STEWARD_POOL_NAME "${FCLI_POOL_NAME}"
ENV FCLI_STEWARD_WALLET_KEY "${FCLI_IMPORT_WALLET_KEY}"
ENV FCLI_STEWARD_SEED "000000000000000000000000Steward1"

RUN echo '[[ ! -z "$STARTUP_FILE_STORAGE_S3" ]] && /s3-copy $STARTUP_FILE_STORAGE_S3 agent /' >/start.sh && \
  echo './import-wallet.sh' >>/start.sh && \
  echo 'exec /findy-agent agency start' >>/start.sh && chmod a+x /start.sh

ENTRYPOINT ["/bin/bash", "-c", "/start.sh"]
