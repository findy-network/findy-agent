#!/bin/bash

getLedgerStatus(){
    local resCode=$(curl -s --write-out '%{http_code}' --output /dev/null http://localhost:9000/genesis)
    if (( ${resCode} == 200 )); then
        return 0
    else
        return 1
    fi
}


NOW=${SECONDS}
printf "Wait until ledger is up"
while ! getLedgerStatus; do
    printf "."
    waitTime=$(($SECONDS - $NOW))
    if (( ${waitTime} >= 60 )); then
        printf "\nLedger failed to start.\n"
        exit 1
    fi
    sleep 1
done
