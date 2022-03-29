#!/bin/bash

if [ "$1" = "host.docker.internal" ]; then echo "127.0.0.1"; else echo "$1"; fi
