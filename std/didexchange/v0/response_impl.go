package v0

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

var responseCreator = &responseFactor{}

type responseFactor struct{}

func (f *responseFactor) NewMsg(init didcomm.MsgInit) didcomm.MessageHdr {
	r := &Response{
		Type:   init.Type,
		ID:     init.AID,
		Thread: &decorator.Thread{ID: init.Nonce},
	}
	return &responseImpl{Response: r}
}

func (f *responseFactor) NewMessage(data []byte) didcomm.MessageHdr {
	return newResponseMsg(data)
}

func init() {
	gob.Register(&responseImpl{})
	aries.Creator.Add(pltype.AriesConnectionResponse, responseCreator)
	aries.Creator.Add(pltype.DIDOrgAriesConnectionResponse, responseCreator)
}

func newResponse(r *Response, ourDID core.DID) (impl *responseImpl, err error) {
	defer err2.Returnf(&err, "new response %s", r.ID)

	r.ConnectionSignature = try.To1(newConnectionSignature(r.Connection, ourDID))

	return &responseImpl{Response: r}, nil
}

func newResponseMsg(data []byte) *responseImpl {
	var mImpl responseImpl
	dto.FromJSON(data, &mImpl)
	mImpl.checkThread()

	mImpl.Connection = try.To1(connectionFromSignedData(mImpl.ConnectionSignature)) // TODO: return/store err

	return &mImpl
}

func (m *responseImpl) checkThread() {
	m.Response.Thread = decorator.CheckThread(m.Response.Thread, m.Response.ID)
}

type responseImpl struct {
	*Response
}

func (m *responseImpl) Thread() *decorator.Thread {
	return m.Response.Thread
}

func (m *responseImpl) ID() string {
	return m.Response.ID
}

func (m *responseImpl) SetID(id string) {
	m.Response.ID = id
}

func (m *responseImpl) Type() string {
	return m.Response.Type
}

func (m *responseImpl) SetType(t string) {
	m.Response.Type = t
}

func (m *responseImpl) JSON() []byte {
	return dto.ToJSONBytes(m)
}

func (m *responseImpl) FieldObj() interface{} {
	return m.Response
}

func (m *responseImpl) Did() string {
	return m.Connection.DID
}

func (m *responseImpl) VerKey() string {
	vms := common.VMs(m.Connection.DIDDoc)
	if len(vms) == 0 {
		return ""
	}
	return base58.Encode(vms[0].Value)
}

func (m *responseImpl) Label() string {
	return ""
}

func (m *responseImpl) DIDDocument() core.DIDDoc {
	return m.Connection.DIDDoc
}

func (m *responseImpl) RoutingKeys() []string {
	return common.Service(m.Connection.DIDDoc, 0).RoutingKeys
}

func (m *responseImpl) Verify(c crypto.Crypto, keyManager kms.KeyManager) error {
	return m.verifySignature(c)
}

func (m *responseImpl) Endpoint() service.Addr {
	services := common.Services(m.Connection.DIDDoc)
	if len(services) == 0 {
		return service.Addr{}
	}

	addr := services[0].ServiceEndpoint
	key := services[0].RecipientKeys[0]

	return service.Addr{Endp: addr, Key: key}
}

func (m *responseImpl) Next(_ string, _ core.DID) (didcomm.Payload, psm.SubState, error) {
	// we are ready at this end for this protocol
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	return aries.PayloadCreator.NewMsg(m.Response.Thread.ID, pltype.AriesConnectionResponse, emptyMsg), psm.ReadyACK, nil
}

func (m *responseImpl) Wait() (didcomm.Payload, psm.SubState) {
	return try.To2(m.Next("", nil))
}

func connectionFromSignedData(cs *ConnectionSignature) (c *Connection, err error) {
	defer err2.Returnf(&err, "connection from signature")

	data := try.To1(utils.DecodeB64(cs.SignedData))
	if len(data) == 0 {
		s := "missing or invalid signature data"
		glog.Error(s)
		return nil, fmt.Errorf(s)
	}
	connectionJSON := data[8:]

	var connection Connection
	dto.FromJSON(connectionJSON, &connection)

	rawDID := common.ID(connection.DIDDoc)
	connection.DID = strings.TrimPrefix(rawDID, "did:sov:")

	return &connection, nil
}

func (m *responseImpl) verifySignature(verifier crypto.Crypto) (err error) {
	defer err2.Returnf(&err, "verify sign")

	data := try.To1(utils.DecodeB64(m.Response.ConnectionSignature.SignedData))
	if len(data) == 0 {
		s := "missing or invalid signature data"
		glog.Error(s)
		return fmt.Errorf(s)
	}

	signature := try.To1(utils.DecodeB64(m.Response.ConnectionSignature.Signature))

	did := ssi.NewDid("", m.Response.ConnectionSignature.SignVerKey)
	try.To(verifier.Verify(signature, data, did.SignKey()))

	timestamp, ok := verifyTimestamp(data)
	if !ok {
		// don't pollute logs with errors when we aren't treating this as an
		// error for now
		glog.Warningln("connection signature timestamp is invalid: ", timestamp, time.Unix(timestamp, 0))
		// TODO: pass invalid timestamps on for now, as some agents do not fill it at all
		// should be fixed with new signature implementation
		// return nil, nil
	} else {
		glog.V(3).Info("verified connection signature w/ ts:", time.Unix(timestamp, 0))
	}

	return nil
}

func verifyTimestamp(data []byte) (timestamp int64, valid bool) {
	const connectionSigExpTime = 10 * 60 * 60

	now := time.Now().Unix()
	tsIsValid := func(ts int64) bool {
		diff := now - ts
		return diff >= 0 && diff <= connectionSigExpTime
	}

	// preferred is big endian
	timestamp = int64(binary.BigEndian.Uint64(data))
	if tsIsValid(timestamp) {
		return timestamp, true
	}

	glog.Warningf("big endian encoded signature timestamp %s is invalid, try little endian", time.Unix(timestamp, 0))

	// accept also meaningful values found in little endian encoding
	// TODO: required format missing from spec
	// => confirm if we should support only preferred big endian
	timestamp = int64(binary.LittleEndian.Uint64(data))
	return timestamp, tsIsValid(timestamp)
}

func newConnectionSignature(connection *Connection, ourDID core.DID) (cs *ConnectionSignature, err error) {
	defer err2.Returnf(&err, "build connection sign")

	connectionJSON := try.To1(json.Marshal(connection))

	signedData, signature, verKey := try.To3(signAndStamp(ourDID, connectionJSON))

	return &ConnectionSignature{
		Type:       "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/signature/1.0/ed25519Sha512_single",
		SignedData: base64.URLEncoding.EncodeToString(signedData),
		SignVerKey: verKey,
		Signature:  base64.URLEncoding.EncodeToString(signature),
	}, nil
}

func getEpochTime() int64 {
	return time.Now().Unix()
}

// SignAndStamp sings and stamps a message and returns the verification key.
// Note! It throws err2 type of error and needs an error handler in the call
// stack.
func signAndStamp(ourDID core.DID, src []byte) (data, dst []byte, vk string, err error) {
	defer err2.Return(&err)

	now := getEpochTime()

	data = make([]byte, 8+len(src))
	binary.BigEndian.PutUint64(data[0:], uint64(now))

	l := copy(data[8:], src)
	if l != len(src) {
		glog.Warning("WARNING, NOT all bytes copied")
	}

	c := ourDID.Packager().Crypto()
	kms := ourDID.Packager().KMS()

	kh := try.To1(kms.Get(ourDID.KID()))
	dst = try.To1(c.Sign(src, kh))
	return data, dst, ourDID.VerKey(), nil
}
