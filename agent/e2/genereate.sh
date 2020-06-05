#!/bin/bash

# README: generates all error wrappers. Note! overwrites old ones.

echo This will overwrite all of the error wrappers.
read -p "Are you sure? " -n 1 -r
echo    # (optional) move to a new line
if [[ $REPLY =~ ^[Yy]$ ]]
then
	go run ../../../../lainio/err2/cmd/main.go -name=Msg -package=e2 -type=mesg.Msg | goimports > msg.go
	go run ../../../../lainio/err2/cmd/main.go -name=Pipe -package=e2 -type=sec.Pipe | goimports > sec.go
	go run ../../../../lainio/err2/cmd/main.go -name=Public -package=e2 -type=endp.Public | goimports > public.go
	go run ../../../../lainio/err2/cmd/main.go -name=Payload -package=e2 -type=*mesg.Payload | goimports > payload.go
	go run ../../../../lainio/err2/cmd/main.go -name=M -package=e2 -type=didcomm.Msg | goimports > m.go
  go run ../../../../lainio/err2/cmd/main.go -name=PL -package=e2 -type=didcomm.Payload | goimports > pl.go
fi

