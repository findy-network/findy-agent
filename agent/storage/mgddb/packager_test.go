package mgddb_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/agent/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
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
	agent           = new(ssi.DIDAgent)
	afgoTestStorage api.AgentStorage
)

func setUp() {
	// first, create agent 1 with the storages
	walletID := fmt.Sprintf("packager-test-agent-1%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()

	agent.OpenWallet(*aw)

	apiStorage := agent.ManagedWallet().Storage()
	afgoTestStorage = apiStorage
}

func TestPackAndUnpack(t *testing.T) {
	ourVdr, err := vdr.New(afgoTestStorage)
	require.NoError(t, err)
	require.NotEmpty(t, ourVdr)

	packager, err := mgddb.NewPackagerFromStorage(afgoTestStorage, ourVdr.Registry())
	require.NoError(t, err)
	require.NotEmpty(t, packager)

	mediaType := transport.MediaTypeProfileDIDCommAIP1

	_, fromPkBytes, err := afgoTestStorage.KMS().CreateAndExportPubKeyBytes(kms.ED25519)
	require.NoError(t, err)
	fromDIDKey, _ := fingerprint.CreateDIDKey(fromPkBytes)

	_, toPkBytes, err := afgoTestStorage.KMS().CreateAndExportPubKeyBytes(kms.ED25519)
	require.NoError(t, err)
	toDIDKey, _ := fingerprint.CreateDIDKey(toPkBytes)

	msg := []byte("msg")
	resBytes, err := packager.PackMessage(&transport.Envelope{
		MediaTypeProfile: mediaType,
		Message:          msg,
		FromKey:          []byte(fromDIDKey),
		ToKeys:           []string{toDIDKey},
	})
	require.NoError(t, err)
	require.NotEmpty(t, resBytes)
	require.NotEqual(t, msg, resBytes)

	resEnvelope, err := packager.UnpackMessage(resBytes)
	require.NoError(t, err)
	require.NotEmpty(t, resEnvelope)
	require.Equal(t, msg, resEnvelope.Message)
}
