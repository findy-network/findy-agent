#!/bin/bash

if [ -z "$FINDY_AGENCY_STEWARD_WALLET_NAME" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_STEWARD_WALLET_NAME"
  exit 1
fi

if [ -z "$FINDY_AGENCY_STEWARD_WALLET_KEY" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_STEWARD_WALLET_KEY"
  exit 1
fi

if [ -z "$FINDY_AGENCY_STEWARD_WALLET_IMPORTED_KEY" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_STEWARD_WALLET_IMPORTED_KEY"
  exit 1
fi

if [ -z "$FINDY_AGENCY_STEWARD_DID" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_STEWARD_DID"
  exit 1
fi

if [ -z "$FINDY_AGENCY_SALT" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_SALT"
  exit 1
fi

if [ -z "FINDY_AGENCY_HOST_ADDRESS" ]; then
  echo "ERROR: Define env variable FINDY_AGENCY_HOST_ADDRESS"
  exit 1
fi

params=(
  "\"findy-agency-steward-wallet-key\":\"$FINDY_AGENCY_STEWARD_WALLET_KEY\""
  "\"findy-agency-steward-wallet-imported-key\":\"$FINDY_AGENCY_STEWARD_WALLET_IMPORTED_KEY\""
  "\"findy-agency-steward-did\":\"$FINDY_AGENCY_STEWARD_DID\""
  "\"findy-agency-salt\":\"$FINDY_AGENCY_SALT\""
  "\"findy-agency-host-address\":\"$FINDY_AGENCY_HOST_ADDRESS\""
)
joined=$(printf ",%s" "${params[@]}")
SECRET_STRING={${joined:1}}

aws secretsmanager create-secret --name findy-agency --secret-string $SECRET_STRING
