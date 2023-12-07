#!/bin/bash

set -e 

AGENT="$GOPATH/bin/findy-agent"
CURRENT_DIR=$(dirname "$BASH_SOURCE")

cd $CURRENT_DIR/../../ && make install
cd ./scripts/export

if [ -z "$FCLI_POOL_NAME" ]; then
  echo "ERROR: define FCLI_POOL_NAME"
  exit 1
fi

if [ -z "$FCLI_POOL_GENESIS_TXN_FILE" ]; then
  echo "ERROR: define FCLI_POOL_GENESIS_TXN_FILE"
  exit 1
fi

if [ -z "$FCLI_STEWARD_SEED" ]; then
  echo "ERROR: define FCLI_STEWARD_SEED"
  exit 1
fi

KEY=$($AGENT tools key create)
FILE="./steward.exported"

$AGENT ledger pool create
$AGENT ledger steward create \
    --pool-name "$FCLI_POOL_NAME" \
    --wallet-key "$KEY" \
    --wallet-name steward_wallet
$AGENT tools export \
    --file "$FILE" \
    --key "$KEY" \
    --wallet-key "$KEY" \
    --wallet-name steward_wallet

echo "Exported steward wallet to $FILE with key $KEY"

