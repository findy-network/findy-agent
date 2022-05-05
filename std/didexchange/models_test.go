package didexchange

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/method"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
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
            "publicKeyBase58": "8KLQJNs7cJFY5vcRTWzb33zYr5zhDrcaX6jgD5Uaofcu"
          }
        ],
        "authentication": [
          {
            "type": "Ed25519SignatureAuthentication2018",
            "publicKey": [
		    "did:sov:ERYihzndieTdh4UA7Q6Y3C#1"
		  ]
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
		{"w3c sample", args{"./json/w3c-doc-sample.json"}, true},
		{"sov from afgo", args{"./json/sov.json"}, true},
		{"our peer did doc", args{"./json/our-peer-did-doc.json"}, true},
		{"acapy 160", args{"json/160-acapy.json"}, false},
		{"acapy", args{"json/acapy.json"}, false},
		{"afgo def", args{"json/afgo-default.json"}, false},
		{"afgo interop", args{"json/afgo-interop.json"}, false},
		{"dotnet", args{"json/dotnet.json"}, true},
		{"findy", args{"json/findy.json"}, false},
		{"js", args{"json/javascript.json"}, false},
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
	tests := []struct {
		name   string
		method method.Type
	}{
		{"sov method", method.TypeSov},
		{"peer method", method.TypePeer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			didIn := agent.NewDID(tt.method, "")
			require.NotNil(t, didIn)

			nonce := "NONCE"
			ae := service.Addr{
				Endp: "http://www.address.com",
				Key:  "SERVICE_KEY",
			}
			didDoc := didIn.NewDoc(ae).(*did.Doc)

			msg := NewRequest(&Request{
				Label: "TestLabel",
				Connection: &Connection{
					DID:    "CALLER_DID",
					DIDDoc: didDoc,
				},
				Thread: &decorator.Thread{ID: nonce},
			})

			opl := aries.PayloadCreator.NewMsg(
				nonce, pltype.AriesConnectionRequest, msg)
			oplJSON := opl.JSON()

			ipl := aries.PayloadCreator.NewFromData(oplJSON)
			iplJSON := ipl.JSON()

			require.Equal(t, oplJSON, iplJSON)

			require.Equal(t, ipl.Type(), pltype.AriesConnectionRequest)

			req := ipl.MsgHdr().FieldObj().(*Request)
			require.NotNil(t, req)
		})
	}
}

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	home := utils.IndyBaseDir()
	removeFiles(home, "/.indy_client/wallet/pipe-test-agent*")
}

func removeFiles(home, nameFilter string) {
	filter := filepath.Join(home, nameFilter)
	files, _ := filepath.Glob(filter)
	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			panic(err)
		}
	}
}

var (
	agent, agent2 = new(ssi.DIDAgent), new(ssi.DIDAgent)
)

func setUp() {
	// init pipe package, TODO: try to find out how to get media profile
	// from...
	sec.Init(transport.MediaTypeProfileDIDCommAIP1)

	// first, create agent 1 with the storages
	walletID := fmt.Sprintf("pipe-test-agent-11%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()
	agent.OpenWallet(*aw)

	// second, create agent 2 with the storages
	walletID2 := fmt.Sprintf("pipe-test-agent-12%d", time.Now().Unix())
	aw2 := ssi.NewRawWalletCfg(walletID2, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw2.Create()
	agent2.OpenWallet(*aw2)
}
