package sec2

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
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

	didIn := a.NewDID("")
	didOut := a.NewDID("")
	didRoute1 := a.NewDID("")
	didRoute2 := a.NewDID("")

	require.NotNil(t, didIn)
	require.NotNil(t, didOut)
	require.NotNil(t, didRoute1)
	require.NotNil(t, didRoute2)

}
