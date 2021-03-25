#!/bin/bash

if [ -z "$STARTUP_FILE_STORAGE_S3" ]; then
  # routines for EB environment bootup
  if [ -f "/tmp/env" ]; then
    echo "Adding environment variables from configuration file."
    eval $(cat /tmp/env)
    cp /tmp/genesis_transactions /genesis_transactions
    cp /tmp/steward.exported /steward.exported
    cp /tmp/aps.p12 /aps.p12
  fi
else
  echo "Copying conf files from s3"
  aws s3 cp s3://$STARTUP_FILE_STORAGE_S3/agent / --recursive
  aws s3 cp s3://$STARTUP_FILE_STORAGE_S3/grpc /grpc --recursive
fi


FOLDER=~/.indy_client/wallet/$FCLI_IMPORT_WALLET_NAME/
if [ -d "$FOLDER" ]; then
  echo "$FOLDER exists"
else
  echo "$FOLDER does not exist, importing wallet"
  ./findy-agent ledger pool create
  ./findy-agent tools import
  echo "{}" > /root/findy.json
fi

cd $1
./findy-agent agency start --grpc true --grpc-cert-path /grpc
