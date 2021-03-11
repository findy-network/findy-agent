#!/bin/bash
# dev.sh

CLI=$GOPATH/bin/findy-agent-cli

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

set -e

clean() {
  echo -e "${GREEN}*** dev - clean ***${NC}"
  echo -e "${RED}WARNING: erasing all local data stored by indy!${NC}"
  rm -rf ~/.indy_client/
  echo "{}" >findy.json
  set +e
  rm findy.bolt
  docker stop findy-pool
  docker rm findy-pool
  docker volume rm sandbox
  set -e
}

run() {
  LEDGER_NAME=$1

  # run agency
  echo -e "${GREEN}*** dev - run agency ***${NC}"
  if [ "$LEDGER_NAME" != "FINDY_FILE_LEDGER" ]; then
    docker start findy-pool
  fi
  $CLI agency start \
    --pool-name ${LEDGER_NAME} \
    --grpc true \
    --steward-wallet-name sovrin_steward_wallet \
    --steward-wallet-key 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --steward-did Th7MpTaRZVRYnPiabds81Y
}

scratch() {
  CURRENT_DIR=$(dirname "$BASH_SOURCE")
  LEDGER_NAME=$1

  # remove and reset all stored data
  clean

  # install latest version of findy-agency
  make install

  # launch and create pool
  if [ "$LEDGER_NAME" != "FINDY_FILE_LEDGER" ]; then
    echo -e "${GREEN}*** dev - start dev ledger ***${NC}"
    docker run -itd -p 9701-9708:9701-9708 \
      -p 9000:9000 \
      -v sandbox:/var/lib/indy/sandbox/ \
      --name findy-pool \
      optechlab/indy-pool-browser:latest
  fi

  echo -e "${GREEN}*** dev - create pool ***${NC}"
  $CLI ledger pool create \
    --name ${LEDGER_NAME} \
    --genesis-txn-file $CURRENT_DIR/genesis_transactions
  echo -e "${GREEN}*** dev - create steward ***${NC}"
  $CLI ledger steward create \
    --pool-name ${LEDGER_NAME} \
    --seed 000000000000000000000000Steward1 \
    --wallet-name sovrin_steward_wallet \
    --wallet-key 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  run $LEDGER_NAME
}

install_run() {
  make install

  run $1
}

onboard() {
  make install

  echo -e "${GREEN}*** dev - onboard ***${NC}"

  # example
  # ./scripts/dev/dev.sh onboard myName 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY .
  EXPORT_NAME=$1
  EXPORT_KEY=$2
  EXPORT_DIR=$3
  echo "name: $EXPORT_NAME, key: $EXPORT_KEY, dir: $EXPORT_DIR"
  set +e
  rm $EXPORT_DIR/${EXPORT_NAME}.export
  rm -rf ~/.indy_client/wallet/${EXPORT_NAME}_client
  rm -rf ~/.indy_client/wallet/${EXPORT_NAME}_server
  set -e
  $CLI service onboard \
    --wallet-name ${EXPORT_NAME}_client \
    --wallet-key ${EXPORT_KEY} \
    --email ${EXPORT_NAME}_server \
    --export-key ${EXPORT_KEY} \
    --agency-url http://localhost:8080 \
    --salt "this is only example" \
    --export-file ${EXPORT_DIR}/${EXPORT_NAME}.export
}

"$@"
