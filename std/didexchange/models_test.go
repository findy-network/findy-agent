package didexchange

import (
	"reflect"
	"testing"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/did"
	"github.com/findy-network/findy-wrapper-go/dto"
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
            "type": "Ed25519SignatureAuthentication2018",
            "publicKey": "did:sov:ERYihzndieTdh4UA7Q6Y3C#1"
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

func TestConnection_ReadJSON(t *testing.T) {
	var req Request

	dto.FromJSONStr(connectionRequest, &req)
	if req.ID != "670bc804-2c06-453c-aee6-48d3c929b488" {
		t.Errorf("id (%v) not match", req.ID)
	}

	doc := req.Connection.DIDDoc
	if doc == nil {
		t.Fail()
		return
	}

	if doc.Authentication == nil ||
		doc.Authentication[0].Type != "Ed25519SignatureAuthentication2018" {
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
			DIDDoc: did.NewDoc(caller, ae),
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
