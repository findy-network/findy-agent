package didexchange1

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/sov/did"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

type commonData struct {
	DID    string
	DIDDoc *decorator.Attachment
}

type commonImpl struct {
	commonData
}

func newDIDDocAttach(_ string, didDoc []byte) *decorator.Attachment {
	data := decorator.AttachmentData{
		Base64: base64.StdEncoding.EncodeToString(didDoc)}
	attachment := &decorator.Attachment{
		ID:       utils.UUID(),
		MimeType: "application/json",
		Data:     data,
	}
	return attachment
}

func (m *commonImpl) Did() string {
	glog.V(11).Infoln("Did() is returning:", m.commonData.DID)
	rawDID := strings.TrimPrefix(m.commonData.DID, "did:sov:")
	if rawDID != m.commonData.DID {
		glog.V(3).Infoln("+++ normalizing Did()", m.commonData.DID, " ==>", rawDID)
	}
	return rawDID
}

func (m *commonImpl) DIDDocument() (coreDoc core.DIDDoc, err error) {
	defer err2.Returnf(&err, "request DID doc")
	assert.NotNil(m.commonData.DIDDoc)

	var doc did.Doc
	didDocBytes := try.To1(base64.StdEncoding.DecodeString(m.commonData.DIDDoc.Data.Base64))
	try.To(json.Unmarshal(didDocBytes, &doc))
	return &doc, nil
}

func (m *commonImpl) VerKey() (key string, err error) {
	defer err2.Returnf(&err, "request VerKey")
	doc := try.To1(m.DIDDocument())
	vm := common.VM(doc, 0)
	return base58.Encode(vm.Value), nil
}

func (m *commonImpl) Endpoint() (addr service.Addr, err error) {
	defer err2.Returnf(&err, "request Endpoint")

	assert.NotNil(m)
	doc := try.To1(m.DIDDocument())

	if len(common.Services(doc)) == 0 {
		return service.Addr{}, nil
	}

	serv := common.Service(doc, 0)
	endp := serv.ServiceEndpoint
	key := serv.RecipientKeys[0]

	return service.Addr{Endp: endp, Key: key}, nil
}
