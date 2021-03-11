#!/bin/bash
# e2e-test.sh

CLI=$GOPATH/bin/findy-agent-cli

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
  echo "{}" >findy.json
  set +e
  rm findy.bolt
  docker stop findy-pool
  docker rm findy-pool
  docker volume rm sandbox
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
  # start dev ledger
  dev_ledger
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

dev_ledger() {
  echo -e "${BLUE}*** dev - start dev ledger ***${NC}"
  docker run -itd -p 9701-9708:9701-9708 \
    -p 9000:9000 \
    -v sandbox:/var/lib/indy/sandbox/ \
    --name findy-pool \
    optechlab/indy-pool-browser:latest
}

unset_envs(){
  unset "${!FCLI@}"
}

set_envs() {
    export FCLI_POOL_NAME="findy"
    export FCLI_POOL_GENESIS_TXN_FILE="${CURRENT_DIR}/genesis_transactions"

    export FCLI_STEWARD_POOL_NAME="findy"
    export FCLI_STEWARD_SEED="000000000000000000000000Steward1"
    export FCLI_STEWARD_WALLET_NAME="sovrin_steward_wallet"
    export FCLI_STEWARD_WALLET_KEY="9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY"

    export FCLI_AGENCY_POOL_NAME="findy"
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

  # onboard
  echo -e "${BLUE}*** env - onboard ***${NC}"
  $CLI service onboard

  # ping
  echo -e "${BLUE}*** env - ping ***${NC}"
  $CLI service ping

  # create schema
  echo -e "${BLUE}*** env - create schema ***${NC}"
  sID=$($CLI service schema create)

  # read schema
  export FCLI_SCHEMA_ID=$sID
  echo -e "${BLUE}*** env - read schema ***${NC}"
  $CLI service schema read

  # create creddef
  export FCLI_CREDDEF_SCHEMA_ID=$sID
  echo -e "${BLUE}*** env - create creddef ***${NC}"
  cID=$($CLI service creddef create)

  # read creddef
  export FCLI_CREDDEF_ID=$cID
  echo -e "${BLUE}*** env - read creddef ***${NC}"
  $CLI service creddef read

  # create key
  echo -e "${BLUE}*** env - create key ***${NC}"
  key=$($CLI tools key create)
  echo $key

  # user onboard
  export FCLI_USER_WALLET_KEY=$key
  unset FCLI_ONBOARD_EXPORT_FILE
  echo -e "${BLUE}*** env - user onboard ***${NC}"
  $CLI user onboard --email=user_test_email1

  # invitation & connect
  echo -e "${BLUE}*** env - invitation & connect ***${NC}"
  conID=$($CLI user invitation | $CLI service connect - | sed 's#^.*}##'| sed -e 's#^.*\[\(.*\)\] ready#\1#' | tr -d '\n')

  # trustping
  echo -e "${BLUE}*** env - trustping ***${NC}"
  export FCLI_TRUSTPING_CONNECTION_ID=$conID
  $CLI service trustping

  # send message
  echo -e "${BLUE}*** env - send message ***${NC}"
  export FCLI_SEND_CONNECTION_ID=$conID
  $CLI service send
}

cmds_conf() {
  unset_envs

  # ping agency
  echo -e "${BLUE}*** conf - ping agency ***${NC}"
  $CLI agency ping --config ${CURRENT_DIR}/configs/agencyPing.yaml

  # onboard
  echo -e "${BLUE}*** conf - onboard ***${NC}"
  $CLI service onboard \
    --config=${CURRENT_DIR}/configs/onboard.yaml \
    --export-file=${CURRENT_DIR}/test_wallet2.export

  # ping
  echo -e "${BLUE}*** conf - ping ***${NC}"
  $CLI service ping --config=${CURRENT_DIR}/configs/ping.yaml

  # create schema
  echo -e "${BLUE}*** conf - create schema ***${NC}"
  sID=$($CLI service schema create --config=${CURRENT_DIR}/configs/createSchema.yaml | sed 's#^.*yaml##' | tr -d '\n')

  # read schema
  echo -e "${BLUE}*** conf - read schema ***${NC}"
  $CLI service schema read --config=${CURRENT_DIR}/configs/service.yaml --id=$sID

  # create creddef
  echo -e "${BLUE}*** conf - create creddef ***${NC}"
  cID=$($CLI service creddef create --schema-id=$sID --config=${CURRENT_DIR}/configs/createCreddef.yaml | sed 's#^.*yaml##' | tr -d '\n')

  # read creddef
  echo -e "${BLUE}*** conf - read creddef ***${NC}"
  $CLI service creddef read --id=$cID --config=${CURRENT_DIR}/configs/service.yaml

  # create key
  echo -e "${BLUE}*** conf - create key ***${NC}"
  key=$($CLI tools key create --config=${CURRENT_DIR}/configs/key.yaml | sed 's#^.*yaml##' | tr -d '\n')
  echo $key

  # user onboard
  echo -e "${BLUE}*** conf - user onboard ***${NC}"
  $CLI user onboard --wallet-key=$key --config=${CURRENT_DIR}/configs/user.yaml

  # invitation & connect
  echo -e "${BLUE}*** conf - invitation & connect ***${NC}"
  conID=$($CLI user invitation --wallet-key=$key --config=${CURRENT_DIR}/configs/user.yaml | sed 's#^.*yaml##' | tr -d '\n' |
  $CLI service connect --config=${CURRENT_DIR}/configs/service.yaml - |
  sed 's#^.*}##'| sed -e 's#^.*\[\(.*\)\] ready#\1#' | sed 's#^.*yaml##' | tr -d '\n')

  # trustping
  echo -e "${BLUE}*** conf - trustping ***${NC}"
  $CLI service trustping --connection-id=$conID --config=${CURRENT_DIR}/configs/service.yaml

  # send message
  echo -e "${BLUE}*** conf - send message ***${NC}"
  $CLI service send --connection-id=$conID --config=${CURRENT_DIR}/configs/service.yaml
}

cmds_flag() {
  unset_envs

  # ping agency
  echo -e "${BLUE}*** flag - ping agency ***${NC}"
  $CLI agency ping --base-address=http://localhost:8090

  # onboard
  echo -e "${BLUE}*** flag - onboard ***${NC}"
  $CLI service onboard \
    --export-file=${CURRENT_DIR}/test_wallet3.export \
    --export-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --agency-url=http://localhost:8090 \
    --wallet-name=test_wallet3 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --email=test_email3 \
    --salt=my_test_salt

  # ping
  echo -e "${BLUE}*** flag - ping ***${NC}"
  $CLI service ping \
  --wallet-name=test_wallet3 \
  --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
  --agency-url=http://localhost:8090

  # create schema
  echo -e "${BLUE}*** flag - create schema ***${NC}"
  sID=$($CLI service schema create \
    --wallet-name=test_wallet3 \
    --agency-url=http://localhost:8090 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --name=my_schema3 --version="2.0" --attributes="["field1", "field2", "field3"]")

  # read schema
  echo -e "${BLUE}*** flag - read schema ***${NC}"
  $CLI service schema read \
    --id=$sID \
    --wallet-name=test_wallet3 \
    --agency-url=http://localhost:8090 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  # create creddef
  echo -e "${BLUE}*** flag - create creddef ***${NC}"
  cID=$($CLI service creddef create \
    --wallet-name=test_wallet3 \
    --agency-url=http://localhost:8090 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --tag=my_tag2 \
    --schema-id=$sID)

  # read creddef
  echo -e "${BLUE}*** flag - read creddef ***${NC}"
  $CLI service creddef read \
    --wallet-name=test_wallet3 \
    --agency-url=http://localhost:8090 \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --id=$cID

  # create key
  echo -e "${BLUE}*** flag - create key ***${NC}"
  key=$($CLI tools key create --seed=000000000000000000000000Steward1)
  echo $key

  # user onboard
  echo -e "${BLUE}*** flag - user onboard ***${NC}"
  $CLI user onboard \
   --wallet-name=user_test_wallet3 \
   --wallet-key=$key \
   --agency-url=http://localhost:8090 \
   --email=user_test_email3 \
   --salt=my_test_salt

  # invitation & connect
  echo -e "${BLUE}*** flag - invitation & connect ***${NC}"
  conID=$($CLI user invitation --label=my_invitation --wallet-name=user_test_wallet3 --wallet-key=$key --agency-url=http://localhost:8090 |
    $CLI service connect --wallet-name=test_wallet3 --agency-url=http://localhost:8090 --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY - |
    sed 's#^.*}##'| sed -e 's#^.*\[\(.*\)\] ready#\1#' | tr -d '\n')

  # trustping
  echo -e "${BLUE}*** flag - trustping ***${NC}"
  $CLI service trustping \
   --wallet-name=test_wallet3 \
   --agency-url=http://localhost:8090 \
   --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
   --connection-id=$conID

  # send message
  echo -e "${BLUE}*** flag - send message ***${NC}"
  $CLI service send \
   --wallet-name=test_wallet3 \
   --agency-url=http://localhost:8090 \
   --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
   --connection-id=$conID --from=me --msg=Hello
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
  $CLI ledger pool ping --name=findy

  echo -e "${BLUE}*** flag - create steward ***${NC}"
  $CLI ledger steward create \
    --pool-name=findy \
    --seed=000000000000000000000000Steward1 \
    --wallet-name=sovrin_steward_wallet \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  # run agency
  echo -e "${BLUE}*** flag - run agency ***${NC}"
  $CLI agency start \
    --pool-name=findy \
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
  $CLI ledger pool ping --name=findy

  echo -e "${BLUE}*** other - import wallet ***${NC}"
  $CLI tools import \
    --key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY \
    --file=${CURRENT_DIR}/steward.exported \
    --wallet-name=steward \
    --wallet-key=9C5qFG3grXfU9LodHdMop7CNVb3HtKddjgRc7oK5KhWY

  # run agency
  echo -e "${BLUE}*** other - run agency ***${NC}"
  $CLI agency start \
    --pool-name=findy \
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
