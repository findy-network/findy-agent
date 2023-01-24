#!/bin/bash

set -e

# TODO: create proper healthcheck tool to agency
# if functionality is needed in the container itself
apt-get update && apt-get install -y curl
curl -f http://localhost:$FCLI_AGENCY_SERVER_PORT/ready || exit 1
