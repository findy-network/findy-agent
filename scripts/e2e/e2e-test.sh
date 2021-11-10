#!/bin/bash
# e2e-test.sh

CLI=$GOPATH/bin/findy-agent

WALLET_KEY='9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY'

CURRENT_DIR=$(dirname "$BASH_SOURCE")

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[1;94m'
BICYAN='\033[1;96m' 
NC='\033[0m'

set -e

e2e() { 
  agency_conf
  agency_flag
  agency_env
  other_cases
  rm_wallets
  echo -e "${BICYAN}*** E2E TEST FINISHED ***${NC}"
}

test_cmds() {
  rm_wallets
  cmds_flag
  cmds_conf
  cmds_env
}

clean() {
  echo -e "${BLUE}*** dev - clean ***${NC}"
  echo -e "${RED}WARNING: erasing all local data stored by indy!${NC}"
  rm -rf ~/.indy_client/
  set +e
  rm findy.bolt
  set -e
}

stop_agency() {
  kill -9 $(ps aux | pgrep 'findy-agent')
}

init_agency(){
  echo -e "${BLUE}*** dev - init agency ***${NC}"
  echo -e "${RED}WARNING: erasing all local data stored by indy!${NC}"
  set +e
  rm -rf ~/.indy_client/
  rm findy.bolt
  set -e
}

init_ledger() {
  # remove and reset all stored data
  clean
}

rm_wallets() {
  set +e
  rm ${CURRENT_DIR}/steward.export.tmp
  set -e
}

unset_envs(){
  unset "${!FCLI@}"
}

set_envs() {
    export FCLI_POOL_NAME="FINDY_FILE_LEDGER"
    export FCLI_POOL_GENESIS_TXN_FILE="${CURRENT_DIR}/genesis_transactions"

    export FCLI_STEWARD_POOL_NAME="FINDY_FILE_LEDGER"
    export FCLI_STEWARD_SEED="000000000000000000000000Steward1"
    export FCLI_STEWARD_WALLET_NAME="sovrin_steward_wallet"
    export FCLI_STEWARD_WALLET_KEY=""$WALLET_KEY""

    export FCLI_AGENCY_POOL_NAME="FINDY_FILE_LEDGER"
    export FCLI_AGENCY_STEWARD_WALLET_NAME="sovrin_steward_wallet"
    export FCLI_AGENCY_STEWARD_WALLET_KEY=""$WALLET_KEY""
    export FCLI_AGENCY_STEWARD_DID="Th7MpTaRZVRYnPiabds81Y"
    export FCLI_AGENCY_STEWARD_SEED="000000000000000000000000Steward1"
    export FCLI_AGENCY_HOST_PORT="8090"
    export FCLI_AGENCY_SERVER_PORT="8090"

    export FCLI_AGENCY_PING_BASE_ADDRESS="http://localhost:8090"

    export FCLI_KEY_SEED="000000000000000000000000Steward1"
}

cmds_env() {
  set_envs

  # ping agency
  echo -e "${BLUE}*** env - ping agency ***${NC}"
  $CLI agency ping

  # create key
  echo -e "${BLUE}*** env - create key ***${NC}"
  key=$($CLI tools key create)
  echo $key
}

cmds_conf() {
  unset_envs

  # ping agency
  echo -e "${BLUE}*** conf - ping agency ***${NC}"
  $CLI agency ping --config ${CURRENT_DIR}/configs/agencyPing.yaml

  # create key
  echo -e "${BLUE}*** conf - create key ***${NC}"
  key=$($CLI tools key create --config=${CURRENT_DIR}/configs/key.yaml | sed 's#^.*yaml##' | tr -d '\n')
  echo $key
}

cmds_flag() {
  unset_envs

  # ping agency
  echo -e "${BLUE}*** flag - ping agency ***${NC}"
  $CLI agency ping --base-address=http://localhost:8090

  # create key
  echo -e "${BLUE}*** flag - create key ***${NC}"
  key=$($CLI tools key create --seed=000000000000000000000000Steward1)
  echo $key
}

agency_env() {
  init_agency
  set_envs

  # launch and create pool
  echo -e "${BLUE}*** env - create pool ***${NC}"
  $CLI ledger pool create
  echo -e "${BLUE}*** env - ping pool ***${NC}"
  $CLI ledger pool ping
  echo -e "${BLUE}*** env - create steward ***${NC}"
  $CLI ledger steward create

  # run agency
  echo -e "${BLUE}*** env - run agency ***${NC}"
  $CLI agency start --config=${CURRENT_DIR}/configs/startAgency.yaml &
  sleep 2
  test_cmds
  stop_agency
}

agency_conf() {
  init_agency
  unset_envs

  # launch and create pool
  echo -e "${BLUE}*** conf - create pool ***${NC}"
  $CLI ledger pool create \
    --config=${CURRENT_DIR}/configs/createPool.yaml \
    --genesis-txn-file=${CURRENT_DIR}/genesis_transactions

  echo -e "${BLUE}*** conf - ping pool ***${NC}"
  $CLI ledger pool ping \
    --config=${CURRENT_DIR}/configs/createPool.yaml

  echo -e "${BLUE}*** conf - create steward ***${NC}"
  $CLI ledger steward create \
    --config=${CURRENT_DIR}/configs/createSteward.yaml

  # run agency
  echo -e "${BLUE}*** conf - run agency ***${NC}"
  $CLI agency start --config=${CURRENT_DIR}/configs/startAgency.yaml &
  sleep 2
  test_cmds
  stop_agency
}

agency_flag() {
  init_agency
  unset_envs

  # launch and create pool
  echo -e "${BLUE}*** flag - create pool ***${NC}"
  $CLI ledger pool create \
    --name=findy \
    --genesis-txn-file=${CURRENT_DIR}/genesis_transactions

  echo -e "${BLUE}*** flag - ping pool ***${NC}"
  $CLI ledger pool ping --name=FINDY_FILE_LEDGER

  echo -e "${BLUE}*** flag - create steward ***${NC}"
  $CLI ledger steward create \
    --pool-name=FINDY_FILE_LEDGER \
    --seed=000000000000000000000000Steward1 \
    --wallet-name=sovrin_steward_wallet \
    --wallet-key="$WALLET_KEY"

  # run agency
  echo -e "${BLUE}*** flag - run agency ***${NC}"
  $CLI agency start \
    --pool-name=FINDY_FILE_LEDGER \
    --steward-wallet-name=sovrin_steward_wallet \
    --steward-wallet-key="$WALLET_KEY" \
    --steward-did=Th7MpTaRZVRYnPiabds81Y \
    --steward-seed=000000000000000000000000Steward1 \
    --host-port=8090 \
    --server-port=8090 \
    --grpc-port=50051 \
    --grpc-cert-path="./grpc/cert" \
    --grpc-jwt-secret="my-secret" &
    sleep 2
  test_cmds
  stop_agency
}

other_cases() {
  init_agency
  unset_envs

  # launch and create pool
  echo -e "${BLUE}*** other - create pool ***${NC}"
  $CLI ledger pool create \
    --name=findy \
    --genesis-txn-file=${CURRENT_DIR}/genesis_transactions

  echo -e "${BLUE}*** other - ping pool ***${NC}"
  $CLI ledger pool ping --name=FINDY_FILE_LEDGER

  echo -e "${BLUE}*** other - import wallet ***${NC}"
  $CLI tools import \
    --key="$WALLET_KEY" \
    --file=${CURRENT_DIR}/steward.exported \
    --wallet-name=steward \
    --wallet-key="$WALLET_KEY"

  # run agency
  echo -e "${BLUE}*** other - run agency ***${NC}"
  $CLI agency start \
    --pool-name=FINDY_FILE_LEDGER \
    --steward-wallet-name=steward \
    --steward-wallet-key="$WALLET_KEY" \
    --steward-did=Th7MpTaRZVRYnPiabds81Y \
    --host-port=8090 \
    --server-port=8090 \
    --grpc-cert-path=./grpc/cert &

  # export
  echo -e "${BLUE}*** other - export wallet ***${NC}"
  f=${CURRENT_DIR}/steward.export.tmp
  $CLI tools export \
    --wallet-name=steward \
    --file=$f \
    --key="$WALLET_KEY" \
    --wallet-key="$WALLET_KEY"
  # check if file exist
  if [ ! -f "$f" ]; then
    echo "$f does not exist."
    exit 1
  fi

  # run agency - no steward

  echo -e "${BLUE}*** other - run agency - no steward ***${NC}"
  stop_agency
  sleep 2
  $CLI agency start \
    --pool-name=FINDY_FILE_LEDGER \
    --steward-wallet-name="" \
    --steward-wallet-key="" \
    --steward-did="" \
    --host-port=8090 \
    --server-port=8090 \
    --grpc-cert-path=./grpc/cert &
  sleep 2
  curl -f localhost:8090



  stop_agency
}
"$@"
