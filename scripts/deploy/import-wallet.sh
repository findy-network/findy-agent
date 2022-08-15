#!/bin/bash

FOLDER=~/.indy_client/pool/$FCLI_POOL_NAME/
if [ -d "$FOLDER" ]; then
  echo "$FOLDER exists"
else
  echo "Creating ledger handle..."
  [[ ! -z "$FCLI_POOL_GENESIS_TXN_FILE" ]] && ./findy-agent ledger pool create
fi

if [ -z "$FCLI_AGENCY_STEWARD_DID" ]; then
  echo "Skipping wallet import as steward is not configured."
  exit 0
fi

FOLDER=~/.indy_client/wallet/$FCLI_AGENCY_STEWARD_WALLET_NAME/
if [ -d "$FOLDER" ]; then
  echo "$FOLDER exists"
else
  if [ -z "$FCLI_IMPORT_WALLET_NAME" ]; then
    echo "Skipping wallet import as import wallet name is not configured."
      ./findy-agent ledger steward create
    exit 0
  fi

  echo "$FOLDER does not exist, importing wallet"
  ./findy-agent tools import
fi
