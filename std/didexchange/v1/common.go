package v1

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	our "github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/sov/did"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

func checkThread(thread *our.Thread, PID string) *our.Thread {
	if thread == nil {
		thread = &our.Thread{PID: PID}
	}
	thread.ID = utils.UUID()
	thread.PID = PID
	return thread
}

type commonData struct {
	DID    string
	DIDDoc *decorator.Attachment
}

type commonImpl struct {
	commonData
}

func newDIDDocAttach(ourDID core.DID) (attachment *decorator.Attachment, err error) {
	defer err2.Handle(&err, "new v1 did doc attachment")

	didDocBytes := try.To1(json.Marshal(ourDID.DOC()))
	data := decorator.AttachmentData{
		Base64: base64.StdEncoding.EncodeToString(didDocBytes)}
	attachment = &decorator.Attachment{
		ID:       utils.UUID(),
		MimeType: "application/json",
		Data:     data,
	}

	// sign attachment
	c := ourDID.Packager().Crypto()
	kms := ourDID.Packager().KMS()
	kh := try.To1(kms.Get(ourDID.KID()))

	b58Key := ourDID.VerKey()
	pubKeyBytes := try.To1(base58.Decode(b58Key))
	pubKey := ed25519.PublicKey(pubKeyBytes)
	try.To(attachment.Data.Sign(c, kh, pubKey, pubKeyBytes))

	return attachment, nil
}

func (m *commonImpl) Did() string {
	glog.V(11).Infoln("Did() is returning:", m.commonData.DID)
	rawDID := strings.TrimPrefix(m.commonData.DID, "did:sov:")
	if rawDID != m.commonData.DID {
		glog.V(3).Infoln("+++ normalizing Did()", m.commonData.DID, " ==>", rawDID)
	}
	return rawDID
}

func (m *commonImpl) DIDDocument() (coreDoc core.DIDDoc) {
	var doc did.Doc
	didDocBytes := try.To1(base64.StdEncoding.DecodeString(m.commonData.DIDDoc.Data.Base64))
	try.To(json.Unmarshal(didDocBytes, &doc))
	return &doc
}

func (m *commonImpl) VerKey() (key string) {
	doc := m.DIDDocument()
	vm := common.VM(doc, 0)
	return base58.Encode(vm.Value)
}

func (m *commonImpl) Endpoint() (addr service.Addr) {
	doc := m.DIDDocument()

	if len(common.Services(doc)) == 0 {
		return service.Addr{}
	}

	serv := common.Service(doc, 0)
	endp := serv.ServiceEndpoint
	key := serv.RecipientKeys[0] // TODO: convert did:key

	return service.Addr{Endp: endp, Key: key}
}

func (m *commonImpl) RoutingKeys() []string {
	doc := m.DIDDocument()
	return common.Service(doc, 0).RoutingKeys // TODO: convert did:key
}

func (m *commonImpl) Verify(DID core.DID) error {
	return m.DIDDoc.Data.Verify(DID.Packager().Crypto(), DID.Packager().KMS())
}
