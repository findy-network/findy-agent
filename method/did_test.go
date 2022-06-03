package method_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/method"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/lainio/err2/try"
	"github.com/stretchr/testify/require"
)

func TestPeer_DIDString(t *testing.T) {
	tests := []struct {
		name   string
		method method.Type
		result string
	}{
		{"peer method", method.TypePeer, "did:peer:"},
		{"key method", method.TypeKey, "did:key:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.method.DIDString(), tt.result)
		})
	}
}

func TestPeer_String(t *testing.T) {
	tests := []struct {
		name   string
		method method.Type
		useKey bool
	}{
		{"peer method with its doc", method.TypePeer, true},
		{"peer method wit build doc", method.TypePeer, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			didIn, _ := agent.NewDID(tt.method, "https://www.address.com")
			require.NotNil(t, didIn)

			var docBytes []byte
			if tt.useKey {
				docBytes = try.To1(json.Marshal(didIn.DOC()))
			} else {
				doc, err := method.NewDoc(didIn.VerKey(), "https://www.address.com")
				require.NoError(t, err)
				require.NotNil(t, doc)
				docBytes = try.To1(json.Marshal(doc))
			}
			out, err := agent.NewOutDID(didIn.URI(), string(docBytes))
			require.NoError(t, err)
			require.NotNil(t, out)
			require.Equal(t, didIn.VerKey(), out.VerKey())
			require.Equal(t, didIn.URI(), out.URI())
		})
	}
}

func TestPeer_Route(t *testing.T) {
	tests := []struct {
		name   string
		method method.Type
	}{
		{"peer method", method.TypePeer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			didIn, _ := agent.NewDID(tt.method, "https://www.address.com")
			require.NotNil(t, didIn)

			route := didIn.Route()
			require.NotNil(t, route)
			require.Len(t, route, 0)
			require.Len(t, didIn.RecipientKeys(), 1)
		})
	}
}

func TestMethodString(t *testing.T) {
	tests := []struct {
		did, method string
	}{
		{did: "did:key",
			method: "key"},
		{did: "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			method: "key"},
		{did: "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso#",
			method: "key"},
		{did: "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso:test#",
			method: "key"},
		{did: "did:sov:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			method: "sov"},
		{did: "did:indy:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			method: "indy"},
	}

	for i, tt := range tests {
		name := fmt.Sprintf("test_%d", i)
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.method, method.String(tt.did))
		})
	}
}

func TestDIDType(t *testing.T) {
	tests := []struct {
		name, did string
		method.Type
	}{
		{"did key only prefix",
			"did:key",
			method.TypeKey,
		},
		{"did key",
			"did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso",
			method.TypeKey,
		},
		{"did peer",
			"did:peer:1zQmQSLFWySB3LACeSrUpvM48QN9frMayNHypnsQjk4GhQKG",
			method.TypePeer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.Type, method.DIDType(tt.did))
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
	walletID := fmt.Sprintf("pipe-test-agent-21%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()
	agent.OpenWallet(*aw)

	// second, create agent 2 with the storages
	walletID2 := fmt.Sprintf("pipe-test-agent-22%d", time.Now().Unix())
	aw2 := ssi.NewRawWalletCfg(walletID2, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw2.Create()
	agent2.OpenWallet(*aw2)
}
