#!/bin/bash

DOCKERHOST=$(./get-docker-host.sh)
if [ "$DOCKERHOST" = "host.docker.internal" ]; then echo "127.0.0.1"; else echo "$DOCKERHOST"; fi
