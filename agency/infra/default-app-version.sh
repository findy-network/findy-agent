#!/bin/bash

aws cloudformation list-exports | grep findy-agency-init-env-envbeanstalka | sed 's/^[ \t]*"Value": "\(.*\)",/\1/'
