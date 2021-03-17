#!/bin/bash

# routines for EB environment bootup
if [ -f "/tmp/env" ]; then
  echo "Adding environment variables from configuration file."
  eval $(cat /tmp/env)
  cp /tmp/genesis_transactions /genesis_transactions
  cp /tmp/steward.exported /steward.exported
  cp /tmp/aps.p12 /aps.p12
fi

if [ -z "$STARTUP_FILE_STORAGE_S3" ]; then
  aws s3 cp s3://$STARTUP_FILE_STORAGE_S3/agent / --recursive
fi


FOLDER=~/.indy_client/wallet/$FCLI_IMPORT_WALLET_NAME/
if [ -d "$FOLDER" ]; then
  echo "$FOLDER exists"
else
  echo "$FOLDER does not exist, importing wallet"
  ./findy-agent ledger pool create
  ./findy-agent tools import
fi

cd $1
./findy-agent agency start --grpc true
