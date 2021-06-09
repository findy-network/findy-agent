#!/bin/bash

check_bash_version() {
	local major=${1:-4}
	local minor=$2
	local rc=0
	local num_re='^[0-9]+$'

	if [[ ! $major =~ $num_re ]] || [[ $minor && ! $minor =~ $num_re ]]; then
		printf '%s\n' "ERROR: version numbers should be numeric"
		return 1
	fi
	if [[ $minor ]]; then
		local bv=${BASH_VERSINFO[0]}${BASH_VERSINFO[1]}
		local vstring=$major.$minor
		local vnum=$major$minor
	else
		local bv=${BASH_VERSINFO[0]}
		local vstring=$major
		local vnum=$major
	fi
	((bv < vnum)) && {
		printf '%s\n' "Warning: Need Bash version $vstring or above, your version is ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"
		rc=1
	}
	return $rc
}

if [[ $_ == $0 ]]; then
	printf "Usage:\tsource ""$0"
	printf " [cli-cmd] [cli-alias]"

	printf "\n\nWhere:\tcli-cmd = fa is default\n"
	printf "\tcli-alias = cli-cmd is default\n"
	exit 1
fi

if [ "$1" = "" ]; then
	CLI=fa
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

# Bash version 3.2 (atleast) has a bug/featuer:
#  https://lists.gnu.org/archive/html/bug-bash/2006-01/msg00018.html

if check_bash_version 3 3; then 
	myCmd=". <(""$CLI"" completion bash | sed 's/findy-agent/$CLI2/g')"
else
	# fallback for e.g. OSX which have old bash
	printf "using workaround for bash process substitution\n"
	sedCmd="$CLI"" completion bash | sed 's/findy-agent/$CLI2/g'"
	myCmd="source /dev/stdin <<<\"\$(""$sedCmd"")\""
	unset sedCmd
fi

eval $myCmd

unset CLI
unset CLI2
unset platform
unset myCmd

