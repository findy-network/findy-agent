package mgddb_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/agent/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	home := utils.IndyBaseDir()
	removeFiles(home, "/.indy_client/wallet/packager-test-agent*")
}

func removeFiles(home, nameFilter string) {
	filter := filepath.Join(home, nameFilter)
	files, _ := filepath.Glob(filter)
	for _, f := range files {
		try.To(os.RemoveAll(f))
	}
}

var (
	agent = new(ssi.DIDAgent)
)

func setUp() {
	// first, create agent 1 with the storages
	walletID := fmt.Sprintf("packager-test-agent-1%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()

	agent.OpenWallet(*aw)
}

func TestPackAndUnpack(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	ourVdr, err := vdr.New(agent.Storage())
	assert.NoError(err)
	assert.NotNil(ourVdr)

	packager, err := mgddb.NewPackagerFromStorage(agent.Storage(), ourVdr.Registry())
	assert.NoError(err)
	assert.NotNil(packager)

	mediaType := transport.MediaTypeProfileDIDCommAIP1

	_, fromPkBytes, err := agent.Storage().KMS().CreateAndExportPubKeyBytes(kms.ED25519)
	assert.NoError(err)
	fromDIDKey, _ := fingerprint.CreateDIDKey(fromPkBytes)

	_, toPkBytes, err := agent.Storage().KMS().CreateAndExportPubKeyBytes(kms.ED25519)
	assert.NoError(err)
	toDIDKey, _ := fingerprint.CreateDIDKey(toPkBytes)

	msg := []byte("msg")
	resBytes, err := packager.PackMessage(&transport.Envelope{
		MediaTypeProfile: mediaType,
		Message:          msg,
		FromKey:          []byte(fromDIDKey),
		ToKeys:           []string{toDIDKey},
	})
	assert.NoError(err)
	assert.SNotEmpty(resBytes)
	assert.NotDeepEqual(msg, resBytes)

	resEnvelope, err := packager.UnpackMessage(resBytes)
	assert.NoError(err)
	assert.NotNil(resEnvelope)
	assert.DeepEqual(msg, resEnvelope.Message)
}
