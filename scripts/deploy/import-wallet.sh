#!/bin/bash

# ignore errors here: no harm in trying ledger creation with each startup
set +e
echo "Creating ledger handle..."
[[ ! -z "$FCLI_POOL_GENESIS_TXN_FILE" ]] && ./findy-agent ledger pool create
set -e

if [ -z "$FCLI_AGENCY_STEWARD_DID" ]; then
  echo "Skipping wallet import as steward is not configured."
  exit 0
fi

FOLDER=~/.indy_client/wallet/$FCLI_IMPORT_WALLET_NAME/
if [ -d "$FOLDER" ]; then
  echo "$FOLDER exists"
else
  echo "$FOLDER does not exist, importing wallet"
  ./findy-agent tools import
fi
