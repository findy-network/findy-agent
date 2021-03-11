#!/bin/bash

# routines for EB environment bootup
if [ -f "/tmp/env" ]; then
  echo "Adding environment variables from configuration file."
  eval $(cat /tmp/env)
  cp /tmp/genesis_transactions /genesis_transactions
  cp /tmp/steward.exported /steward.exported
  cp /tmp/aps.p12 /aps.p12
fi

FOLDER=~/.indy_client/wallet/$FCLI_IMPORT_WALLET_NAME/
if [ -d "$FOLDER" ]; then
  echo "$FOLDER exists"
else
  echo "$FOLDER does not exist, importing wallet"
  ./findy-agent-cli ledger pool create
  ./findy-agent-cli tools import
fi

cd $1
./findy-agent-cli agency start --grpc true
