#!/bin/bash

FOLDER=~/.indy_client/wallet/$FCLI_IMPORT_WALLET_NAME/
if [ -d "$FOLDER" ]; then
  echo "$FOLDER exists"
else
  echo "$FOLDER does not exist, importing wallet"
  ./findy-agent ledger pool create
  ./findy-agent tools import
  echo "{}" > /root/findy.json
fi
