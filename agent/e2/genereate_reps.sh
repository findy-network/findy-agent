#!/bin/bash

# README: generates all error wrappers. Note! overwrites old ones.

echo This will overwrite all of the error wrappers.
read -p "Are you sure? " -n 1 -r
echo    # (optional) move to a new line
if [[ $REPLY =~ ^[Yy]$ ]]
then
	go run ../../../../lainio/err2/cmd/main.go -name=PresentProofRep -package=e2 -type=*psm.PresentProofRep | goimports > present_proof_rep.go
	go run ../../../../lainio/err2/cmd/main.go -name=IssueCredRep -package=e2 -type=*psm.IssueCredRep | goimports > issue_cred_rep.go
	go run ../../../../lainio/err2/cmd/main.go -name=BasicMessageRep -package=e2 -type=*psm.BasicMessageRep | goimports > basic_message_rep.go
	go run ../../../../lainio/err2/cmd/main.go -name=PSM -package=e2 -type=*psm.PSM | goimports > psm.go
fi

