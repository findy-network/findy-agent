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
	agent, agent2 = new(ssi.DIDAgent), new(ssi.DIDAgent)

	pckr, pckr2 *packager.Packager

	agentStorage, agentStorage2 *mgddb.Storage
)

func setUp() {
	Init(transport.MediaTypeProfileDIDCommAIP1)

	walletID := fmt.Sprintf("pipe-test-agent-1%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()

	agent.OpenWallet(*aw)

	apiStorage := agent.ManagedWallet().Storage()
	agentStorage = apiStorage.(*mgddb.Storage)

	pckr = try.To1(packager.New(agentStorage, agent.VDR().Registry()))

	walletID2 := fmt.Sprintf("pipe-test-agent-2%d", time.Now().Unix())
	aw2 := ssi.NewRawWalletCfg(walletID2, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw2.Create()

	agent2.OpenWallet(*aw2)

	apiStorage2 := agent2.ManagedWallet().Storage()
	agentStorage2 = apiStorage2.(*mgddb.Storage)

	pckr2 = try.To1(packager.New(agentStorage2, agent2.VDR().Registry()))
}

func TestNewPipe(t *testing.T) {
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

	p := Pipe{Pckr: pckr, In: didIn, Out: didOut}

	packed, _ := try.To2(p.Pack(message))
	received, _ := try.To2(p.Unpack(packed))
	require.Equal(t, message, received)

	// pipe sign/verify
	sign, _ := p.Sign(message)
	_, _ = p.Verify(message, sign)

	//require.True(t, ok)
}

func TestResolve(t *testing.T) {
	vdr := agent.VDR() // .Registry()
	docR := try.To1(vdr.Registry().Resolve(key1))
	require.NotNil(t, docR)
	bytes := try.To1(docR.DIDDocument.JSONBytes())
	require.NotNil(t, bytes)
}

func TestPackTowardsPubKeyOnly(t *testing.T) {
	didIn := agent.NewDID("key")
	require.NotNil(t, didIn)
	println(didIn.String())
	didOut := agent.NewOutDID(key2)
	require.NotNil(t, didOut)
	println(didOut.String())

	message := []byte("message")

	p := Pipe{Pckr: pckr, In: didIn, Out: didOut}

	packed, _ := try.To2(p.Pack(message))
	require.NotNil(t, packed)
}

func TestSignVerify(t *testing.T) {
	didIn2 := agent2.NewDID("key")
	require.NotNil(t, didIn2)
	println("in2: ", didIn2.String())

	didIn := agent.NewDID("key")
	require.NotNil(t, didIn)
	println("in: ", didIn.String())

	didOut := agent.NewOutDID(didIn2.String())
	require.NotNil(t, didOut)
	println("out: ", didOut.String())

	didOut2 := agent2.NewOutDID(didIn.String())
	require.NotNil(t, didOut2)
	println("out2: ", didOut2.String())

	message := []byte("message")

	p := Pipe{Pckr: pckr, In: didIn, Out: didOut}
	p2 := Pipe{Pckr: pckr2, In: didIn2, Out: didOut2}

	packed, _ := try.To2(p.Pack(message))
	require.NotNil(t, packed)
	received, _ := try.To2(p2.Unpack(packed))
	require.Equal(t, message, received)

	// Note!! We cannot unpack packed message because we don't have private key
	// here any more, we have only pre generated public keys to test signing

	// We still can test verify() because it's done towards public key
	// pipe.Sign sign message towards Out DID
	sign, _ := p.Sign(message)
	ok, _ := p2.Verify(message, sign)

	require.True(t, ok)
}
