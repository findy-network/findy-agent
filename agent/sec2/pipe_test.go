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

func setUp() {
}

func TestNewPipe(t *testing.T) {
	walletID := fmt.Sprintf("pipe-test-agent-%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()

	a := ssi.DIDAgent{}
	a.OpenWallet(*aw)

	apiStorage := a.ManagedWallet().Storage()
	as, ok := apiStorage.(*mgddb.Storage)
	require.True(t, ok, "todo: update type later!!")

	pckr := try.To1(packager.New(as, a.VDR().Registry()))
	require.NotNil(t, pckr)
	Init(pckr, transport.MediaTypeProfileDIDCommAIP1)

	didIn := a.NewDID("key")
	didOut := a.NewDID("key")
	didRoute1 := a.NewDID("key")
	didRoute2 := a.NewDID("key")

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
	ok, _ = p.Verify(message, sign)

	require.True(t, ok)

}
