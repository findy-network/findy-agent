package didexchange

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/stretchr/testify/assert"
)

// Connection request taken from Python Agent output for example json.
var connectionRequest = `  {
    "@type": "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/request",
    "@id": "670bc804-2c06-453c-aee6-48d3c929b488",
    "label": "Alice Agent",
    "connection": {
      "DID": "ERYihzndieTdh4UA7Q6Y3C",
      "DIDDoc": {
        "@context": "https://w3id.org/did/v1",
        "id": "did:sov:ERYihzndieTdh4UA7Q6Y3C",
        "publicKey": [
          {
            "id": "did:sov:ERYihzndieTdh4UA7Q6Y3C#1",
            "type": "Ed25519VerificationKey2018",
            "controller": "did:sov:ERYihzndieTdh4UA7Q6Y3C",
            "publicKeyBase58": "8KLQJNs7cJFY5vcRTWzb33zYr5zhDrcaX6jgD5Uaofcu"
          }
        ],
        "authentication": [
          {
            "id": "did:sov:ERYihzndieTdh4UA7Q6Y3C#1",
            "type": "Ed25519SignatureAuthentication2018",
            "publicKey": "did:sov:ERYihzndieTdh4UA7Q6Y3C#1",
            "controller": "did:sov:ERYihzndieTdh4UA7Q6Y3C"
          }
        ],
        "service": [
          {
            "id": "did:sov:ERYihzndieTdh4UA7Q6Y3C;indy",
            "type": "IndyAgent",
            "priority": 0,
            "recipientKeys": ["8KLQJNs7cJFY5vcRTWzb33zYr5zhDrcaX6jgD5Uaofcu"],
            "serviceEndpoint": "http://192.168.65.3:8030"
          }
        ]
      }
    }
  }
`

var didDocStr = `{
  "@context": [
    "https://www.w3.org/ns/did/v1"
  ],
  "id": "did:peer:1zQmZkgq3q4eCXrKqV6DqR1DcADaC8vPgW1b2m4QambYjiMz",
  "verificationMethod": [
    {
      "controller": "",
      "id": "GPrfyh84g4qtKk4VyPihJuK5KSuTy_5apWYOTzE62to",
      "publicKeyBase58": "6yQAVAebciDaprMxhStV3GpxqRP6d7JoeEApFvm56u3s",
      "type": "Ed25519VerificationKey2018"
    }
  ],
  "service": [
    {
      "id": "didcomm",
      "priority": 0,
      "recipientKeys": [
        "6yQAVAebciDaprMxhStV3GpxqRP6d7JoeEApFvm56u3s"
      ],
      "routingKeys": [],
      "serviceEndpoint": "http://example.com",
      "type": "did-communication"
    }
  ],
  "authentication": [
    {
      "controller": "",
      "id": "GPrfyh84g4qtKk4VyPihJuK5KSuTy_5apWYOTzE62to",
      "publicKeyBase58": "6yQAVAebciDaprMxhStV3GpxqRP6d7JoeEApFvm56u3s",
      "type": "Ed25519VerificationKey2018"
    }
  ]
}`

// test json from service json testing.
var serviceJSON = `{
  "id": "did:sov:ERYihzndieTdh4UA7Q6Y3C;indy",
  "type": "IndyAgent",
  "priority": 3,
  "recipientKeys": ["8KLQJNs7cJFY5vcRTWzb33zYr5zhDrcaX6jgD5Uaofcu"],
  "serviceEndpoint": "http://192.168.65.3:8030"
}`

func TestConnection_ReadServiceJSON(t *testing.T) {
	var s did.Service
	dto.FromJSONStr(serviceJSON, &s)

	if s.ID != "did:sov:ERYihzndieTdh4UA7Q6Y3C;indy" {
		t.Errorf("error in service reading ID = %v", s.ID)
	}

	if len(s.RecipientKeys) == 0 {
		t.Errorf("error in service reading RecipientKeys length 0")
	}
}

func TestConnection_ReadDoc(t *testing.T) {
	err2.StackStraceWriter = os.Stderr
	var doc did.Doc

	err := json.Unmarshal([]byte(didDocStr), &doc)
	assert.NoError(t, err)

}

func TestConnection_ReadJSON(t *testing.T) {
	err2.StackStraceWriter = os.Stderr

	var req Request

	dto.FromJSONStr(connectionRequest, &req)
	if req.ID != "670bc804-2c06-453c-aee6-48d3c929b488" {
		t.Errorf("id (%v) not match", req.ID)
	}

	d := req.Connection.DIDDoc
	if d == nil {
		t.Fail()
		return
	}

	b := try.To1(json.Marshal(d))
	bs := string(b)
	println(bs)

	var doc did.Doc
	try.To(json.Unmarshal(b, &doc))

	if doc.Authentication == nil ||
		doc.Authentication[0].VerificationMethod.Type != "Ed25519SignatureAuthentication2018" {
		t.Errorf("id (%v) not match", doc.Authentication)
	}

	recipKey := doc.Service[0].RecipientKeys[0]
	if recipKey != "8KLQJNs7cJFY5vcRTWzb33zYr5zhDrcaX6jgD5Uaofcu" {
		t.Errorf("id (%v) not match", recipKey)
	}
}

func TestNewRequest(t *testing.T) {
	caller := ssi.NewDid("CALLER_DID", "CALLER_VERKEY")
	nonce := "NONCE"
	ae := service.Addr{Endp: "http://www.address.com", Key: "SERVICE_KEY"}
	msg := NewRequest(&Request{
		Label: "TestLabel",
		Connection: &Connection{
			DID:    "CALLER_DID",
			DIDDoc: caller.NewDoc(ae).(*did.Doc),
		},
		Thread: &decorator.Thread{ID: nonce},
	})
	opl := aries.PayloadCreator.NewMsg(nonce, pltype.AriesConnectionRequest, msg)

	json := opl.JSON()

	ipl := aries.PayloadCreator.NewFromData(json)

	if pltype.AriesConnectionRequest != ipl.Type() {
		t.Errorf("wrong type %v", ipl.Type())
	}

	req := ipl.MsgHdr().FieldObj().(*Request)
	if req == nil {
		t.Error("request is nil")
	}

	if !reflect.DeepEqual(opl, ipl) {
		t.Errorf("not equal, is (%v), want (%v)", req, msg)
	}
}
