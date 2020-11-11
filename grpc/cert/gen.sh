#!/bin/bash

if [ "$1" == "" ]; then
	echo "Usage: ""$0"" <server|client>"
	exit 1
fi

cfg="$1"/cert.conf 
if [ ! -f $cfg ]; then
	echo 'file not found '"$cfg"
fi

openssl genrsa -out "$1"/"$1".key 4096 
openssl req -nodes -new -x509 -sha256 -days 1825 -config "$cfg" -extensions 'req_ext' -key "$1"/"$1".key -out "$1"/"$1".crt

