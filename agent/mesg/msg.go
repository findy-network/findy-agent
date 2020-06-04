package mesg

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/golang/glog"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/sec"
	"github.com/optechlab/findy-agent/agent/service"
	"github.com/optechlab/findy-agent/agent/ssi"
	"github.com/optechlab/findy-agent/agent/utils"
	"github.com/optechlab/findy-agent/std/decorator"
	didexchange "github.com/optechlab/findy-agent/std/didexchange/invitation"
	"github.com/optechlab/findy-go/crypto"
	"github.com/optechlab/findy-go/dto"
)

var MsgCreator = MsgFactor{}

func init() {
	didcomm.CreatorGod.AddMsgCreator(pltype.Agent, MsgCreator)
	didcomm.CreatorGod.AddMsgCreator(pltype.CA, MsgCreator)
	didcomm.CreatorGod.AddMsgCreator(pltype.SA, MsgCreator)
}

type MsgFactor struct{}

func (f MsgFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	return f.Create(init)
}

func (f MsgFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return &MsgImpl{Msg: newMsg(data)}
}

func (f MsgFactor) NewAnonDecryptedMsg(wallet int, cryptStr string, did *ssi.DID) didcomm.Msg {
	m := NewAnonDecryptedMsg(wallet, cryptStr, did)
	return &MsgImpl{Msg: m}
}

func (f MsgFactor) Create(d didcomm.MsgInit) didcomm.MessageHdr {
	m := CreateMsg(d)
	return &MsgImpl{Msg: &m}
}

// CreateMsg is a helper function for creating MsgInits to help constructing
// "wrappers" for Msgs.
func CreateMsg(d didcomm.MsgInit) Msg {
	m := Msg{
		Encrypted:   d.Encrypted,
		Did:         d.Did,
		Nonce:       d.Nonce,
		Error:       d.Error,
		VerKey:      d.VerKey,
		RcvrEndp:    d.RcvrEndp.Endp,
		RcvrKey:     d.RcvrEndp.Key,
		Endpoint:    d.Endpoint,
		EndpVerKey:  d.EndpVerKey,
		Name:        d.Name,
		Info:        d.Info,
		ID:          d.ID,
		Ready:       d.Ready,
		Msg:         d.Msg,
		Body:        d.Body,
		ProofValues: d.ProofValues,
	}
	if d.DIDObj != nil {
		m.Did = d.DIDObj.Did()
		m.VerKey = d.DIDObj.VerKey()
	}
	return m
}

type MsgImpl struct {
	*Msg
}

func (m *MsgImpl) Thread() *decorator.Thread {
	return decorator.NewThread(m.Msg.Nonce, "")
}

func (m *MsgImpl) ID() string {
	panic("not in use in old API messages")
}

func (m *MsgImpl) SetID(id string) {
	panic("not in use in old API messages")
}

func (m *MsgImpl) Type() string {
	panic("not in use in old API messages")
}

func (m *MsgImpl) SetType(t string) {
	panic("not in use in old API messages")
}

func (m *MsgImpl) JSON() []byte {
	panic("not in use in old API messages")
}

func (m *MsgImpl) Error() string {
	return m.Msg.Error
}

func (m *MsgImpl) SetInfo(s string) {
	m.Msg.Info = s
}

func (m *MsgImpl) SubLevelID() string {
	return m.Msg.ID
}

func (m *MsgImpl) SetSubLevelID(s string) {
	m.Msg.ID = s
}

func (m *MsgImpl) Schema() *ssi.Schema {
	return m.Msg.Schema
}

func (m *MsgImpl) SetSchema(sch *ssi.Schema) {
	m.Msg.Schema = sch
}

func (m *MsgImpl) SetReady(yes bool) {
	m.Msg.Ready = yes
}

func (m *MsgImpl) SetBody(b interface{}) {
	m.Msg.Body = b
}

func (m *MsgImpl) SetInvitation(i *didexchange.Invitation) {
	m.Msg.Invitation = i
}

func (m *MsgImpl) SetSubMsg(sm map[string]interface{}) {
	m.Msg.Msg = sm
}

func (m *MsgImpl) SetDid(s string) {
	m.Msg.Did = s
}

func (m *MsgImpl) SetVerKey(s string) {
	m.Msg.VerKey = s
}

func (m *MsgImpl) Ready() bool {
	return m.Msg.Ready
}

func (m *MsgImpl) Info() string {
	return m.Msg.Info
}

func (m *MsgImpl) TimestampMs() *uint64 {
	return m.Msg.TimestampMs
}

func (m *MsgImpl) ConnectionInvitation() *didexchange.Invitation {
	return m.Msg.Invitation
}

func (m *MsgImpl) CredentialAttributes() *[]didcomm.CredentialAttribute {
	return m.Msg.CredentialAttrs
}

func (m *MsgImpl) CredDefID() *string {
	return m.Msg.CredDefID
}

func (m *MsgImpl) ProofAttributes() *[]didcomm.ProofAttribute {
	return m.Msg.ProofAttrs
}

func (m *MsgImpl) ProofValues() *[]didcomm.ProofValue {
	return m.Msg.ProofValues
}

func (m *MsgImpl) SetNonce(n string) {
	m.Msg.Nonce = n
}

func (m *MsgImpl) SetError(s string) {
	m.Msg.Error += s
}

func (m *MsgImpl) Did() string {
	return m.Msg.Did
}

func (m *MsgImpl) VerKey() string {
	return m.Msg.VerKey
}

func (m *MsgImpl) Nonce() string {
	return m.Msg.Nonce
}

func (m *MsgImpl) Encrypted() string {
	return m.Msg.Encrypted
}

func (m *MsgImpl) FieldObj() interface{} {
	return m.Msg
}

func (m *MsgImpl) Name() string {
	return m.Msg.Name
}

func (m *MsgImpl) SubMsg() map[string]interface{} {
	return m.Msg.Msg
}

func (m *MsgImpl) ReceiverEP() service.Addr {
	return service.Addr{
		Endp: m.RcvrEndp,
		Key:  m.RcvrKey,
	}
}

func (m *MsgImpl) Endpoint() service.Addr {
	return service.Addr{
		Endp: m.Msg.Endpoint,
		Key:  m.Msg.EndpVerKey,
	}
}

func (m *MsgImpl) Encr(cp sec.Pipe) didcomm.Msg {
	return &MsgImpl{Msg: m.Msg.Encrypt(cp)}
}

func (m *MsgImpl) Decr(cp sec.Pipe) didcomm.Msg {
	return &MsgImpl{Msg: m.Msg.Decrypt(cp)}
}

func (m *MsgImpl) AnonEncrypt(did *ssi.DID) didcomm.Msg {
	return &MsgImpl{Msg: m.Msg.AnonEncrypt(did)}
}

// The salt used to calculate sha256 checksum of connection invite (Handshake)
// message. Add agent/salt.go to your specific build BUT don't add it to this
// Git repo. We have already ignored it in .gitignore. That's how you can over
// ride quite easily to default salt.
//
//		package agent
//
//		func init() {
//			salt = "THIS IS YOUR PROJECT SPECIFIC SALT"
//		}
//

// The Msg is multipurpose way to transfer messages inside actual Payload
// which will be standardized hopefully by hyperledger-indy or someone.
// The Msg works like C language union i.e. if Error happened is filled
// and others are empty. Very similarly works Encrypted field. Please feel
// free to add new fields if needed.
type Msg struct {
	Error           string                         `json:"error,omitempty"`           // If error happens includes error msg
	Encrypted       string                         `json:"encrypted,omitempty"`       // If the whole msg is encrypted is transferred in this field
	Did             string                         `json:"did,omitempty"`             // Usually senders DID and corresponding VerKey
	Nonce           string                         `json:"nonce,omitempty"`           // Important field to keep track receiving/sending sync
	VerKey          string                         `json:"verkey,omitempty"`          // Senders Verkey for DID
	RcvrEndp        string                         `json:"rcvr_endp,omitempty"`       // Receivers own endpoint, usually the public URL
	RcvrKey         string                         `json:"rcvr_key,omitempty"`        // Receiver endpoint ver key
	Endpoint        string                         `json:"endpoint,omitempty"`        // Multipurpose field which still is under design
	EndpVerKey      string                         `json:"endp_ver_key,omitempty"`    // VerKey associated to endpoint i.e. payload verkey
	Name            string                         `json:"name,omitempty"`            // Multipurpose field which still is under design
	Info            string                         `json:"info,omitempty"`            // Used for transferring additional info like the Msg in IM-cases, and Pairwise name
	TimestampMs     *uint64                        `json:"timestamp,omitempty"`       // Used for transferring timestamp info
	Invitation      *didexchange.Invitation        `json:"invitation,omitempty"`      // Used for connection invitation
	CredentialAttrs *[]didcomm.CredentialAttribute `json:"credAttributes,omitempty"`  // Used for credential attributes values
	CredDefID       *string                        `json:"credDefId,omitempty"`       // Used for credential definition ID
	ProofAttrs      *[]didcomm.ProofAttribute      `json:"proofAttributes,omitempty"` // Used for proof attributes
	ProofValues     *[]didcomm.ProofValue          `json:"proofValues,omitempty"`     // Used for proof values
	ID              string                         `json:"id,omitempty"`              // Used for transferring additional ID like the Cred Def ID
	Ready           bool                           `json:"ready,omitempty"`           // In queries tells if something is ready when true
	Schema          *ssi.Schema                    `json:"schema,omitempty"`          // Schema data for creating schemas for the cred defs
	Msg             map[string]interface{}         `json:"msg,omitempty"`             // Generic sub message to transport JSON between Indy SDK and EAs
	Body            interface{}                    `json:"body,omitempty"`            // Task status data
}

func SubFromJSONData(data []byte) (sub map[string]interface{}) {
	err := json.Unmarshal(data, &sub)
	if err != nil {
		glog.Error("Error marshalling from JSON: ", err.Error())
		return nil
	}
	return
}

func SubFromJSON(s string) (sub map[string]interface{}) {
	return SubFromJSONData([]byte(s))
}

func newMsgFrom(js string) (msg *Msg) {
	if js == "" {
		js = "{}"
	}
	return newMsg([]byte(js))
}

func newMsg(bytes []byte) (msg *Msg) {
	err := json.Unmarshal(bytes, &msg)
	if err != nil {
		glog.Error("Error marshalling from JSON: ", err.Error())
		return nil
	}
	return
}

func NewAnonDecryptedMsg(wallet int, cryptStr string, did *ssi.DID) *Msg {
	f := ssi.Future{}
	msg, err := base64.StdEncoding.DecodeString(cryptStr)
	if err != nil {
		panic(err)
	}
	f.SetChan(crypto.AnonDecrypt(wallet, did.VerKey(), msg))
	msgJSON := f.Bytes()
	return newMsgFrom(string(msgJSON))
}

func NewHandshake(email, pwName string) *Msg {
	checkStr := checksum(email)
	msg := Msg{
		Endpoint: email,
		Name:     pwName,
		Info:     checkStr,
		Nonce:    "0",
	}
	return &msg
}

func checksum(email string) string {
	salted := []byte(email + utils.Salt)
	checkSum := sha256.Sum256(salted)
	checkStr := base64.StdEncoding.EncodeToString(checkSum[:])
	return checkStr
}

func (m *Msg) ChecksumOK() bool {
	return m.Info == checksum(m.Endpoint)
}

func (m *Msg) anonDecrypt(wallet int, did *ssi.DID) *Msg {
	return NewAnonDecryptedMsg(wallet, m.Encrypted, did)
}

func (m *Msg) AnonEncrypt(did *ssi.DID) *Msg {
	mb := dto.ToJSONBytes(m)
	f := ssi.Future{}
	ch := crypto.AnonCrypt(did.VerKey(), mb)
	f.SetChan(ch)
	msgBytes := f.Result().Bytes()
	ec := base64.StdEncoding.EncodeToString(msgBytes)
	return &Msg{Encrypted: ec}
}

func (m *Msg) Decrypt(cp sec.Pipe) *Msg {
	msg, err := base64.StdEncoding.DecodeString(m.Encrypted)
	if err != nil {
		panic(err)
	}
	msgJSON := cp.Decrypt(msg)
	return newMsgFrom(string(msgJSON))
}

func (m *Msg) Encrypt(cp sec.Pipe) *Msg {
	mb := dto.ToJSONBytes(m)
	msgBytes := cp.Encrypt(mb)
	ec := base64.StdEncoding.EncodeToString(msgBytes)
	return &Msg{Encrypted: ec}

}

func (m *Msg) SetEndpoint(ae service.Addr) {
	m.Endpoint = ae.Endp
	m.EndpVerKey = ae.Key
}

func (m *Msg) SetRcvrEndp(ae service.Addr) {
	m.RcvrEndp = ae.Endp
	m.RcvrKey = ae.Key
}
