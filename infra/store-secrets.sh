#!/bin/bash

if [ -z "$FINDY_DEV_LEDGER_STEWARD_WALLET_NAME" ]; then
  echo "ERROR: Define env variable FINDY_DEV_LEDGER_STEWARD_WALLET_NAME"
  exit 1
fi

if [ -z "$FINDY_DEV_LEDGER_STEWARD_WALLET_KEY" ]; then
  echo "ERROR: Define env variable FINDY_DEV_LEDGER_STEWARD_WALLET_KEY"
  exit 1
fi

if [ -z "$FINDY_DEV_LEDGER_STEWARD_WALLET_IMPORTED_KEY" ]; then
  echo "ERROR: Define env variable FINDY_DEV_LEDGER_STEWARD_WALLET_IMPORTED_KEY"
  exit 1
fi

if [ -z "$FINDY_DEV_LEDGER_STEWARD_DID" ]; then
  echo "ERROR: Define env variable FINDY_DEV_LEDGER_STEWARD_DID"
  exit 1
fi

if [ -z "$FINDY_AGENT_SALT" ]; then
  echo "ERROR: Define env variable FINDY_AGENT_SALT"
  exit 1
fi

if [ -z "FINDY_AGENT_HOST_ADDRESS" ]; then
  echo "ERROR: Define env variable FINDY_AGENT_HOST_ADDRESS"
  exit 1
fi

params=(
  "\"findy-agent-dev-ledger-steward-wallet-name\":\"$FINDY_DEV_LEDGER_STEWARD_WALLET_NAME\""
  "\"findy-agent-dev-ledger-steward-wallet-key\":\"$FINDY_DEV_LEDGER_STEWARD_WALLET_KEY\""
  "\"findy-agent-dev-ledger-steward-wallet-imported-key\":\"$FINDY_DEV_LEDGER_STEWARD_WALLET_IMPORTED_KEY\""
  "\"findy-agent-dev-ledger-steward-did\":\"$FINDY_DEV_LEDGER_STEWARD_DID\""
  "\"findy-agent-salt\":\"$FINDY_AGENT_SALT\""
  "\"findy-agent-host-address\":\"$FINDY_AGENT_HOST_ADDRESS\""
)
joined=$(printf ",%s" "${params[@]}")
SECRET_STRING={${joined:1}}

aws secretsmanager create-secret --name findy-agent --secret-string $SECRET_STRING
