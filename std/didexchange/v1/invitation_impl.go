package v1

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
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

var invitationCreator = &invitationFactor{}

type invitationFactor struct{}

func (f *invitationFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	inv, err := invitation.Create(invitation.DIDExchangeVersionV1, invitation.AgentInfo{
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
	aries.Creator.Add(pltype.AriesOutOfBandInvitation, invitationCreator)
	aries.Creator.Add(pltype.DIDOrgAriesOfBandInvitation, invitationCreator)
}

type invitationImpl struct {
	didexchange.UnsupportedPwMsgBase
	invitation.Invitation
	thread *decorator.Thread
}

func newInvitation(inv invitation.Invitation) *invitationImpl {
	return &invitationImpl{Invitation: inv, thread: checkThread(&decorator.Thread{}, inv.ID())}
}

func NewInvitationMsg(data []byte) *invitationImpl {
	dataStr := string(data)
	inv, err := invitation.Translate(dataStr)
	if err != nil {
		glog.Warningf("invitation translation failed %s", dataStr)
		return nil
	}
	impl := newInvitation(inv)
	checkThread(impl.thread, impl.Invitation.ID())
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

func (m *invitationImpl) Next(ourLabel string, ourDID core.DID) (
	pl didcomm.Payload,
	st psm.SubState,
	err error,
) {
	defer err2.Returnf(&err, "next for v1 invitation")

	// build a connection request message to send to another agent
	msg := try.To1(newRequest(ourDID, &Request{
		Label:  ourLabel,
		DID:    ourDID.Did(),
		Thread: checkThread(&decorator.Thread{}, m.thread.PID),
	}))

	// Create payload to send
	return aries.PayloadCreator.NewMsg(
		m.thread.PID,
		pltype.DIDOrgAriesDIDExchangeRequest,
		msg), psm.Sending, nil

}

func (m *invitationImpl) Wait() (didcomm.Payload, psm.SubState) {
	return aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   m.thread.PID,
			Type: pltype.DIDOrgAriesDIDExchangeRequest,
		}), psm.Waiting
}
