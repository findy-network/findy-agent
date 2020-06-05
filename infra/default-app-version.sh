#!/bin/bash

aws cloudformation list-exports | grep findy-agent-init-env-envbeanstalka | sed 's/^[ \t]*"Value": "\(.*\)",/\1/'
