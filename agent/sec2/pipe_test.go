package sec2

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/packager"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/lainio/err2/try"
	"github.com/stretchr/testify/require"
)

const (
	key1 = "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso"
	key2 = "did:key:z6MkqQ81wZSsjWeTk4MnPVow3Jyydp31AP7qNj3WvUtrdejx"
	key3 = "did:key:z6MkmPrHsyXEeujwhpMGSyyxmixpuqUYQ2QPfj3Y3gFPugNp"
	key4 = "did:key:z6MkuMg4H1GH2XdLPuBMcuDvWx18NNHFie37PN37GP7V1L4G"
)

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
	agent        = new(ssi.DIDAgent)
	agentStorage *mgddb.Storage
)

func setUp() {
	walletID := fmt.Sprintf("pipe-test-agent-%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()

	agent.OpenWallet(*aw)

	apiStorage := agent.ManagedWallet().Storage()
	agentStorage = apiStorage.(*mgddb.Storage)

	//	_ = try.To1(packager.New(as, agent.VDR().Registry()))
	//	Init(pckr, transport.MediaTypeProfileDIDCommAIP1)

}

func TestNewPipe(t *testing.T) {
	pckr := try.To1(packager.New(agentStorage, agent.VDR().Registry()))
	require.NotNil(t, pckr)
	Init(pckr, transport.MediaTypeProfileDIDCommAIP1)

	didIn := agent.NewDID("key")
	println(didIn.String())
	didOut := agent.NewDID("key")
	println(didOut.String())
	didRoute1 := agent.NewDID("key")
	println(didRoute1.String())
	didRoute2 := agent.NewDID("key")
	println(didRoute2.String())

	require.NotNil(t, didIn)
	require.NotNil(t, didOut)
	require.NotNil(t, didRoute1)
	require.NotNil(t, didRoute2)

	message := []byte("message")

	p := Pipe{In: didIn, Out: didOut}

	packed, _ := try.To2(p.Pack(message))
	received, _ := try.To2(p.Unpack(packed))
	require.Equal(t, message, received)

	// pipe sign/verify
	sign, _ := p.Sign(message)
	ok, _ := p.Verify(message, sign)

	require.True(t, ok)
}

func TestResolveAndSignVerify(t *testing.T) {
	vdr := agent.VDR() // .Registry()
	docR := try.To1(vdr.Registry().Resolve(key1))
	bytes := try.To1(docR.DIDDocument.JSONBytes())
	println(string(bytes))

	didIn := agent.NewDID("key")
	println(didIn.String())
	didOut := agent.NewOutDID(key2)
	println(didOut.String())

	require.NotNil(t, didIn)
	require.NotNil(t, didOut)

	message := []byte("message")

	p := Pipe{In: didIn, Out: didOut}

	_, _ = try.To2(p.Pack(message))
	//	received, _ := try.To2(p.Unpack(packed))
	//	require.Equal(t, message, received)

	// pipe sign/verify
	sign, _ := p.Sign(message)
	ok, _ := p.Verify(message, sign)

	require.True(t, ok)
}
