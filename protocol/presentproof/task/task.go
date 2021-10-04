package task

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
)

type ProofActionType string

const (
	AcceptRequest ProofActionType = "accept_request"
	AcceptPropose ProofActionType = "accept_propose"
	AcceptValues  ProofActionType = "accept_values"
)

type TaskPresentProof struct {
	comm.TaskBase
	Comment         string
	ProofAttrs      []didcomm.ProofAttribute
	ProofPredicates []didcomm.ProofPredicate
	ActionType      ProofActionType
}
