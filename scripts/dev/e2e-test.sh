#!/bin/bash
# e2e-test.sh

CLI=$GOPATH/bin/findy-agent

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
#  other_cases
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
  echo "{}" >findy.json
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
  echo "{}" >findy.json
  rm findy.bolt
  set -e
}

init_ledger() {
  # remove and reset all stored data
  clean
}

rm_wallets() {
  set +e
  rm ${CURRENT_DIR}/test_wallet1.export
  rm -rf ~/.indy_client/wallet/test_wallet1
  rm -rf ~/.indy_client/wallet/test_email1

  rm -rf ~/.indy_client/wallet/user_test_wallet1
  rm -rf ~/.indy_client/wallet/user_test_email1

  rm -rf ~/.indy_client/wallet/user_test_wallet3
  rm -rf ~/.indy_client/wallet/user_test_email3

  rm -rf ~/.indy_client/wallet/user_test_wallet2
  rm -rf ~/.indy_client/wallet/user_test_email2

  rm ${CURRENT_DIR}/test_wallet2.export
  rm -rf ~/.indy_client/wallet/test_wallet2
  rm -rf ~/.indy_client/wallet/test_email2

  rm ${CURRENT_DIR}/test_wallet3.export
  rm -rf ~/.indy_client/wallet/test_wallet3
  rm -rf ~/.indy_client/wallet/test_email3

  rm ${CURRENT_DIR}/test_wallet4.export
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
    export FCLI_STEWARD_WALLET_KEY="9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY"

    export FCLI_AGENCY_POOL_NAME="FINDY_FILE_LEDGER"
    export FCLI_AGENCY_STEWARD_WALLET_NAME="sovrin_steward_wallet"
    export FCLI_AGENCY_STEWARD_WALLET_KEY="9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY"
    export FCLI_AGENCY_STEWARD_DID="Th7MpTaRZVRYnPiabds81Y"
    export FCLI_AGENCY_STEWARD_SEED="000000000000000000000000Steward1"
    export FCLI_AGENCY_SALT="my_test_salt"
    export FCLI_AGENCY_HOST_PORT="8090"
    export FCLI_AGENCY_SERVER_PORT="8090"

    export FCLI_AGENCY_PING_BASE_ADDRESS="http://localhost:8090"

    export FCLI_SERVICE_WALLET_NAME="test_wallet1"
    export FCLI_SERVICE_WALLET_KEY="9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY"
    export FCLI_SERVICE_AGENCY_URL="http://localhost:8090"
    export FCLI_ONBOARD_EMAIL="test_email1"
    export FCLI_ONBOARD_EXPORT_FILE="${CURRENT_DIR}/test_wallet1.export"
    export FCLI_ONBOARD_EXPORT_KEY="9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY"
    export FCLI_ONBOARD_SALT="my_test_salt"

    export FCLI_FCLI_PING_SERVICE_ENDPOINT="true"

    export FCLI_SCHEMA_NAME="my_schema1"
    export FCLI_SCHEMA_VERSION="2.0"
    export FCLI_SCHEMA_ATTRIBUTES="[\"field1\", \"field2\", \"field3\"]"

    export FCLI_CREDDEF_TAG="my_tag1"

    export FCLI_KEY_SEED="000000000000000000000000Steward1"

    export FCLI_USER_WALLET_NAME="user_test_wallet1"
    export FCLI_USER_AGENCY_URL="http://localhost:8090"

    export FCLI_INVITATION_LABEL="my_label"

    export FCLI_SEND_MESSAGE="Hello!"
    export FCLI_SEND_FROM="me"
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
  $CLI agency start &
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
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  # run agency
  echo -e "${BLUE}*** flag - run agency ***${NC}"
  $CLI agency start \
    --pool-name=FINDY_FILE_LEDGER \
    --steward-wallet-name=sovrin_steward_wallet \
    --steward-wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --steward-did=Th7MpTaRZVRYnPiabds81Y \
    --steward-seed=000000000000000000000000Steward1 \
    --host-port=8090 \
    --server-port=8090 \
    --salt=my_test_salt &
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
    --key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --file=${CURRENT_DIR}/steward.exported \
    --wallet-name=steward \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  # run agency
  echo -e "${BLUE}*** other - run agency ***${NC}"
  $CLI agency start \
    --pool-name=FINDY_FILE_LEDGER \
    --steward-wallet-name=steward \
    --steward-wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --steward-did=Th7MpTaRZVRYnPiabds81Y \
    --steward-seed=000000000000000000000000Steward1 \
    --host-port=8090 \
    --server-port=8090 \
    --salt=this is only example &
    sleep 2

  # onboard
  echo -e "${BLUE}*** other - onboard ***${NC}"
  $CLI service onboard \
    --agency-url=http://localhost:8090 \
    --wallet-name=test_wallet1 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --email=test_email1 \
    --salt=this is only example

  # create schema
  echo -e "${BLUE}*** other - create schema ***${NC}"
  sID=$($CLI service schema create \
    --wallet-name=test_wallet1 \
    --agency-url=http://localhost:8090 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --name=my_schema4 --version="2.0" --attributes="["field1", "field2", "field3"]")

  # read schema
  echo -e "${BLUE}*** other - read schema ***${NC}"
  $CLI service schema read \
    --id=$sID \
    --wallet-name=test_wallet1 \
    --agency-url=http://localhost:8090 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  # export
  echo -e "${BLUE}*** other - export wallet ***${NC}"
  f=${CURRENT_DIR}/test_wallet4.export 
  $CLI tools export \
    --wallet-name=test_wallet1 \
    --file=$f \
    --key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY
  # check if file exist
  if [ ! -f "$f" ]; then
    echo "$f does not exist."
    exit 1
  fi
  stop_agency
}
"$@"
