package task

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
)

type IssueActionType string

const (
	AcceptOffer   IssueActionType = "accept_offer"
	AcceptPropose IssueActionType = "accept_propose"
)

type TaskIssueCredential struct {
	comm.TaskBase
	Comment         string
	CredentialAttrs []didcomm.CredentialAttribute
	CredDefID       string
	ActionType      IssueActionType
}
