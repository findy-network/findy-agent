#!/bin/bash

if [[ $_ == $0 ]]; then
	printf "Usage:\tsource ""$0"
	printf " [cli-cmd] [cli-alias]"

	printf "\n\nWhere:\tcli-cmd = cli is default\n"
	printf "\tcli-alias = cli-cmd is default\n"
	exit 1
fi

if [ "$1" = "" ]; then
  CLI=cli
else
  CLI="$1"
fi
echo "cli is set to:" "$CLI"

if [ "$2" = "" ]; then
  CLI2="$CLI"
else
  CLI2="$2"
fi
echo "used cli alias is:" "$CLI2"

myCmd=". <(""$CLI"" completion bash | sed 's/findy-agent/$CLI2/g')"

eval $myCmd

unset CLI
unset CLI2
unset platform
unset myCmd
