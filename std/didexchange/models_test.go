package didexchange

import (
	"encoding/json"
	"io/ioutil"
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
	"github.com/stretchr/testify/require"
)

// Connection request taken from Python Agent output for example json.
var connectionRequest = `  {
    "@type": "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/request",
    "@id": "670bc804-2c06-453c-aee6-48d3c929b488",
    "label": "Alice Agent",
    "connection": {
      "DID": "ERYihzndieTdh4UA7Q6Y3C",
      "DIDDoc": {
        "@context": "https://www.w3.org/2019/did/v1",
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
            "id": "did:sov:ERYihzndieTdh4UA7Q6Y3C",
            "type": "Ed25519SignatureAuthentication2018",
            "publicKey": [
		    "did:sov:ERYihzndieTdh4UA7Q6Y3C#1"
		  ],
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
	err2.StackTraceWriter = os.Stderr
	defer err2.CatchTrace(func(err error) {
		t.Error(err)
	})

	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args
		ok bool
	}{
		{"w3c sample", args{"./w3c-doc-sample.json"}, false},
		{"sov from afgo", args{"./sov.json"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc did.Doc
			d, err := ioutil.ReadFile(tt.filename)
			require.NoError(t, err)

			if tt.ok {
				require.NoError(t, json.Unmarshal(d, &doc))
			} else {
				require.Error(t, json.Unmarshal(d, &doc))
			}
		})
	}
}

func TestConnection_ReadJSON(t *testing.T) {
	err2.StackTraceWriter = os.Stderr

	var req Request

	err := json.Unmarshal([]byte(connectionRequest), &req)
	require.NoError(t, err)
	require.Equal(t, "670bc804-2c06-453c-aee6-48d3c929b488", req.ID)

	doc := req.Connection.DIDDoc

	require.NotNil(t, doc.Authentication)
	require.Equal(t, "Ed25519VerificationKey2018", doc.Authentication[0].VerificationMethod.Type)

	recipKey := doc.Service[0].RecipientKeys[0]
	require.Equal(t, "8KLQJNs7cJFY5vcRTWzb33zYr5zhDrcaX6jgD5Uaofcu", recipKey)
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
