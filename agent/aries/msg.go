package aries

import (
	"encoding/gob"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/std/decorator"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
)

var MsgCreator = MsgFactor{}

func init() {
	gob.Register(&msgImpl{})
	didcomm.CreatorGod.AddMsgCreator(pltype.Aries, MsgCreator)
	didcomm.CreatorGod.AddMsgCreator(pltype.DIDOrgAries, MsgCreator)
}

type MsgFactor struct{}

func (f MsgFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	m := createMsg(init)
	return &msgImpl{Msg: &m}
}

func (f MsgFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return newMsg(data)
}

func (f MsgFactor) Create(d didcomm.MsgInit) didcomm.MessageHdr {
	factor, ok := Creator.factors[d.Type]
	if !ok {
		m := createMsg(d)
		return &msgImpl{Msg: &m}
	}
	return factor.NewMsg(d)
}

func createMsg(d didcomm.MsgInit) Msg {
	th := d.Thread
	if th == nil {
		th = decorator.NewThread(d.Nonce, "")
	}
	m := Msg{
		AID:        d.AID,
		Type:       d.Type,
		Did:        d.Did,
		Thread:     th,
		Error:      d.Error,
		VerKey:     d.VerKey,
		RcvrEndp:   d.RcvrEndp.Endp,
		RcvrKey:    d.RcvrEndp.Key,
		Endpoint:   d.Endpoint,
		EndpVerKey: d.EndpVerKey,
		Name:       d.Name,
		Info:       d.Info,
		ID:         d.ID,
		Ready:      d.Ready,
		Msg:        d.Msg,
	}
	if d.DIDObj != nil {
		m.Did = d.DIDObj.Did()
		m.VerKey = d.DIDObj.VerKey()
	}
	return m
}

type msgImpl struct {
	*Msg
}

func (m *msgImpl) Thread() *decorator.Thread {
	return m.Msg.Thread
}

func (m *msgImpl) ID() string {
	return m.Msg.AID
}

func (m *msgImpl) SetID(id string) {
	m.Msg.AID = id
}

func (m *msgImpl) Type() string {
	return m.Msg.Type
}

func (m *msgImpl) SetType(t string) {
	m.Msg.Type = t
}

func (m *msgImpl) JSON() []byte {
	return dto.ToJSONBytes(m.Msg)
}

func (m *msgImpl) Error() string {
	return m.Msg.Error
}

func (m *msgImpl) SetInfo(s string) {
	m.Msg.Info = s
}

func (m *msgImpl) SubLevelID() string {
	return m.Msg.ID
}

func (m *msgImpl) SetSubLevelID(s string) {
	m.Msg.ID = s
}

func (m *msgImpl) Schema() *ssi.Schema {
	panic("not in use in old API messages")
}

func (m *msgImpl) SetSchema(sch *ssi.Schema) {
	panic("not in use in old API messages")
}

func (m *msgImpl) SetReady(yes bool) {
	m.Msg.Ready = yes
}

func (m *msgImpl) SetBody(b interface{}) {
	panic("not in use in old API messages")
}

func (m *msgImpl) SetInvitation(i *didexchange.Invitation) {
	panic("not in use in old API messages")
}

func (m *msgImpl) SetSubMsg(sm map[string]interface{}) {
	m.Msg.Msg = sm
}

func (m *msgImpl) SetDid(s string) {
	m.Msg.Did = s
}

func (m *msgImpl) SetVerKey(s string) {
	m.Msg.VerKey = s
}

func (m *msgImpl) Ready() bool {
	return m.Msg.Ready
}

func (m *msgImpl) Info() string {
	return m.Msg.Info
}

func (m *msgImpl) TimestampMs() *uint64 {
	return nil
}

func (m *msgImpl) ConnectionInvitation() *didexchange.Invitation {
	return nil
}

func (m *msgImpl) CredentialAttributes() *[]didcomm.CredentialAttribute {
	return nil
}

func (m *msgImpl) CredDefID() *string {
	return nil
}

func (m *msgImpl) ProofAttributes() *[]didcomm.ProofAttribute {
	return nil
}

func (m *msgImpl) ProofValues() *[]didcomm.ProofValue {
	return nil
}

func (m *msgImpl) SetNonce(n string) {
	if th := m.Msg.Thread; th != nil {
		th.ID = n
		return
	}
	m.Msg.Thread = &decorator.Thread{ID: n}
}

func (m *msgImpl) SetError(s string) {
	m.Msg.Error += s
}

func (m *msgImpl) Did() string {
	return m.Msg.Did
}

func (m *msgImpl) VerKey() string {
	return m.Msg.VerKey
}

func (m *msgImpl) Nonce() string {
	if th := m.Msg.Thread; th != nil {
		return th.ID
	}
	glog.Warning("Returning ID() for nonce/thread_id")
	return m.ID()
}

func (m *msgImpl) Endpoint() service.Addr {
	return service.Addr{
		Endp: m.Msg.Endpoint,
		Key:  m.Msg.EndpVerKey,
	}
}

func (m *msgImpl) FieldObj() interface{} {
	return m.Msg
}

func (m *msgImpl) Name() string {
	return m.Msg.Name
}

func (m *msgImpl) SubMsg() map[string]interface{} {
	return m.Msg.Msg
}

func (m *msgImpl) ReceiverEP() service.Addr {
	return service.Addr{
		Endp: m.RcvrEndp,
		Key:  m.RcvrKey,
	}
}

type Msg struct {
	Type string `json:"@type,omitempty"`
	AID  string `json:"@id,omitempty"`

	Thread *decorator.Thread `json:"~thread,omitempty"`

	Error      string                 `json:"error,omitempty"`        // If error happens includes error msg
	Did        string                 `json:"did,omitempty"`          // Usually senders DID and corresponding VerKey
	VerKey     string                 `json:"verkey,omitempty"`       // Senders Verkey for DID
	RcvrEndp   string                 `json:"rcvr_endp,omitempty"`    // Receivers own endpoint, usually the public URL
	RcvrKey    string                 `json:"rcvr_key,omitempty"`     // Receiver endpoint ver key
	Endpoint   string                 `json:"endpoint,omitempty"`     // Multipurpose field which still is under design
	EndpVerKey string                 `json:"endp_ver_key,omitempty"` // VerKey associated to endpoint i.e. payload verkey
	Name       string                 `json:"name,omitempty"`         // Multipurpose field which still is under design
	Info       string                 `json:"info,omitempty"`         // Used for transferring additional info like the Msg in IM-cases, and Pairwise name
	ID         string                 `json:"id,omitempty"`           // Used for transferring additional ID like the Cred Def ID
	Ready      bool                   `json:"ready,omitempty"`        // In queries tells if something is ready when true
	Msg        map[string]interface{} `json:"msg,omitempty"`          // Generic sub message to transport JSON between Indy SDK and EAs
}

func newMsg(data []byte) *msgImpl {
	var mImpl msgImpl
	dto.FromJSON(data, &mImpl)
	return &mImpl
}

func (m *Msg) SetEndpoint(ae service.Addr) {
	m.Endpoint = ae.Endp
	m.EndpVerKey = ae.Key
}

func (m *Msg) SetRcvrEndp(ae service.Addr) {
	m.RcvrEndp = ae.Endp
	m.RcvrKey = ae.Key
}
