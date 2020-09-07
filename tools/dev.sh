#!/bin/bash
# dev.sh

AGENT=$GOPATH/bin/findy-agent

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
  # run agency
  echo -e "${GREEN}*** dev - run agency ***${NC}"
  $AGENT server \
    -pool findy \
    -wallet sovrin_steward_wallet \
    -pwd 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    -did Th7MpTaRZVRYnPiabds81Y
}

scratch() {
  # remove and reset all stored data
  clean

  # install latest version of findy-agent
  make install

  # launch and create pool
  echo -e "${GREEN}*** dev - launch and create pool ***${NC}"
  docker run -itd -p 9701-9708:9701-9708 \
    -p 9000:9000 \
    -v sandbox:/var/lib/indy/sandbox/ \
    --restart=always \
    --name findy-pool optechlab/indy-pool-browser:latest
  $AGENT create cnx \
    -pool findy \
    -txn $PWD/.circleci/genesis_transactions
  $AGENT create steward \
    -pool findy \
    -seed 000000000000000000000000Steward1 \
    -wallet sovrin_steward_wallet \
    -pwd 9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  run
}

install_run() {
  make install

  run
}

onboard() {
  make install

  echo -e "${GREEN}*** dev - onboard ***${NC}"

  # example
  # ./tools/dev.sh onboard myName QzU2ubQJGy5CqzbuiDnbiLcreZHdJWs7HjZKp4ft2Mx .
  EXPORT_NAME=$1
  EXPORT_KEY=$2
  EXPORT_DIR=$3
  echo "name: $EXPORT_NAME, key: $EXPORT_KEY, dir: $EXPORT_DIR"
  set +e
  rm $EXPORT_DIR/${EXPORT_NAME}.export
  rm -rf ~/.indy_client/wallet/${EXPORT_NAME}_client
  rm -rf ~/.indy_client/wallet/${EXPORT_NAME}_server
  set -e
  $AGENT client handshakeAndExport \
    -wallet ${EXPORT_NAME}_client \
    -email ${EXPORT_NAME}_server \
    -pwd ${EXPORT_KEY} \
    -url http://localhost:8080 \
    -exportpath ${EXPORT_DIR}/${EXPORT_NAME}.export
}

"$@"
