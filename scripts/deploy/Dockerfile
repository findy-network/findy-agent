ARG CLI_VERSION=latest
ARG AWS_VERSION=2.1.30

FROM findy-agent:$CLI_VERSION

ADD ./start-server.sh /start-server.sh

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

ENV FCLI_AGENCY_APNS_P12_FILE ""

ARG AWS_VERSION

RUN apt-get update && apt-get install curl unzip -y && \
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-$AWS_VERSION.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install

ENTRYPOINT [ "/start-server.sh", "/"]