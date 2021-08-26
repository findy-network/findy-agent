package comm

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-agent/std/didexchange/invitation"
)

type Task struct {
	Nonce        string
	TypeID       string // "connection", "issue-credential", "trust_ping"
	SenderEndp   service.Addr
	ReceiverEndp service.Addr
	Message      string
	ID           string
	Info         string

	// Pairwise
	ConnectionInvitation *invitation.Invitation

	// Issue credential
	CredentialAttrs *[]didcomm.CredentialAttribute
	CredDefID       *string

	// Present proof
	ProofAttrs *[]didcomm.ProofAttribute
}

// SwitchDirection changes SenderEndp and ReceiverEndp data
func (t *Task) SwitchDirection() {
	tmp := t.SenderEndp
	t.SenderEndp = t.ReceiverEndp
	t.ReceiverEndp = tmp
}

// NewTaskRawPayload creates a new task from raw PL.
func NewTaskRawPayload(ipl didcomm.Payload) (t *Task) {
	return &Task{
		Nonce:  ipl.ThreadID(),
		TypeID: ipl.Type(),
	}
}

// NewTaskRawPayload creates a new task from raw PL
func NewTaskFromRequest(ipl didcomm.Payload, req *didexchange.Request) (t *Task) {
	senderEP := service.Addr{
		Endp: req.Connection.DIDDoc.Service[0].ServiceEndpoint,
		Key:  req.Connection.DIDDoc.Service[0].RecipientKeys[0],
	}
	return &Task{
		// We take nonce from connection ID which is tranferred with ThreadID
		Nonce:      ipl.ThreadID(),
		TypeID:     ipl.Type(),
		SenderEndp: senderEP,
		// We use same connection ID for pairwise naming
		Message:    ipl.ThreadID(),
	}
}

// NewTaskFromConnectionResponse creates a new task from raw PL
func NewTaskFromConnectionResponse(ipl didcomm.Payload, res *didexchange.Response) (t *Task) {
	return &Task{
		Nonce:        ipl.ThreadID(),
		TypeID:       ipl.Type(),
		SenderEndp:   res.Endpoint(),
		ReceiverEndp: res.Endpoint(),
	}
}
