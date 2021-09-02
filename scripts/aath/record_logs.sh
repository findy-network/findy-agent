#!/bin/bash

CONTAINER_NAME=$1

container_starting(){
    if [ "$( docker container inspect -f '{{.State.Running}}' $CONTAINER_NAME )" == "true" ]; then
        return 0
    else
        return 1
    fi
}


NOW=${SECONDS}
printf "Wait until container is up\n"
while ! container_starting; do
    waitTime=$(($SECONDS - $NOW))
    if (( ${waitTime} >= 600 )); then
        printf "\nContainer failed to start.\n"
        exit 1
    fi
    sleep 1
done

printf "Start saving logs"
docker logs -f $CONTAINER_NAME > .logs/$CONTAINER_NAME-$(date +%s).log
