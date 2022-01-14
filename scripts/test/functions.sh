#!/bin/bash

get_abs_filename() {
	# $1 : relative filename
	echo "$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
}

brew_location() {
	echo $(brew --prefix "$1")
}

prompt_default_yes() {
	read -r -p "Do you want that? [Y/n] " response
	if [[ "$response" =~ ^([nN][oO]|[nN])$ ]]
	then
	    echo no
	else
	    echo yes
	fi
}

prompt_default_no() {
	read -r -p "Are you sure? [y/N] " response
	if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]
	then
	    echo yes
	else
	    echo no
	fi
}

