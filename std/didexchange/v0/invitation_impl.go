package v0

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/findy-network/findy-common-go/std/didexchange/invitation"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
)

var invitationCreator = &invitationFactor{}

type invitationFactor struct{}

func (f *invitationFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	inv, err := invitation.Create(invitation.DIDExchangeVersionV0, invitation.AgentInfo{
		InvitationID:   init.AID,
		InvitationType: init.Type,
	})
	if err != nil {
		glog.Warningf("invitation creation failed %s", err.Error())
		return nil
	}
	return newInvitation(inv)
}

func (f *invitationFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return NewInvitationMsg(data)
}

func init() {
	gob.Register(&invitationImpl{})
	aries.Creator.Add(pltype.AriesConnectionInvitation, invitationCreator)
	aries.Creator.Add(pltype.DIDOrgAriesConnectionInvitation, invitationCreator)
}

type invitationImpl struct {
	didexchange.UnsupportedPwMsgBase
	invitation.Invitation
	thread *decorator.Thread
}

func newInvitation(inv invitation.Invitation) *invitationImpl {
	impl := &invitationImpl{Invitation: inv, thread: &decorator.Thread{}}
	impl.checkThread()
	return impl
}

func (m *invitationImpl) checkThread() {
	m.thread = decorator.CheckThread(m.thread, m.Invitation.ID())
}

func NewInvitationMsg(data []byte) *invitationImpl {
	dataStr := string(data)
	inv, err := invitation.Translate(dataStr)
	if err != nil {
		glog.Warningf("invitation translation failed %s", dataStr)
		return nil
	}
	impl := newInvitation(inv)
	impl.checkThread()
	return impl
}

func (m *invitationImpl) Thread() *decorator.Thread {
	return m.thread
}

func (m *invitationImpl) ID() string {
	return m.Invitation.ID()
}

func (m *invitationImpl) Type() string {
	return m.Invitation.Type()
}

func (m *invitationImpl) JSON() []byte {
	return dto.ToJSONBytes(m)
}

func (m *invitationImpl) FieldObj() interface{} {
	return m.Invitation
}

func (m *invitationImpl) Label() string {
	return m.Invitation.Label()
}

func (m *invitationImpl) VerKey() string {
	return m.Invitation.Services()[0].RecipientKeysAsB58()[0]
}

func (m *invitationImpl) Endpoint() service.Addr {
	return service.Addr{
		Endp: m.Invitation.Services()[0].ServiceEndpoint,
		Key:  m.VerKey(),
	}
}

func (m *invitationImpl) RoutingKeys() []string {
	return m.Invitation.Services()[0].RoutingKeysAsB58()
}

func (m *invitationImpl) Verify(c crypto.Crypto, keyManager kms.KeyManager) error {
	return nil
}

func (m *invitationImpl) Next(ourLabel string, ourDID core.DID) (didcomm.Payload, psm.SubState, error) {
	// build a connection request message to send to another agent
	msg := newRequest(&Request{
		Label: ourLabel,
		Connection: &Connection{
			DID:    ourDID.Did(),
			DIDDoc: ourDID.DOC(),
		},
		// when out-of-bound and did-exchange protocols are supported we
		// should start to save connection_id to Thread.PID
		Thread: &decorator.Thread{ID: m.thread.ID},
	})

	// Create payload to send
	return aries.PayloadCreator.NewMsg(m.thread.ID, pltype.AriesConnectionRequest, msg), psm.Sending, nil

}

func (m *invitationImpl) Wait() (didcomm.Payload, psm.SubState) {
	return aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   m.thread.ID,
			Type: pltype.AriesConnectionRequest,
		}), psm.Waiting
}
