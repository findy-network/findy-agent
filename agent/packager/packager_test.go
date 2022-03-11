package packager

import (
	"flag"
	"os"
	"testing"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/findy-network/findy-agent/agent/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/stretchr/testify/require"
)

var (
	storageTestConfig = api.AgentStorageConfig{
		AgentKey: mgddb.GenerateKey(),
		AgentID:  "agentID",
		FilePath: ".",
	}
	afgoTestStorage *mgddb.Storage
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	err2.Check(flag.Set("logtostderr", "true"))
	err2.Check(flag.Set("stderrthreshold", "WARNING"))
	err2.Check(flag.Set("v", "10"))
	flag.Parse()

	// AFGO
	var err error
	afgoTestStorage, err = mgddb.New(storageTestConfig)
	assert.D.True(err == nil)
	assert.D.True(afgoTestStorage != nil)
}

func tearDown() {
	err := afgoTestStorage.Close()
	assert.D.True(err == nil)

	os.RemoveAll(storageTestConfig.AgentID + ".bolt")
}

func TestPackAndUnpack(t *testing.T) {
	ourVdr, err := vdr.New(afgoTestStorage)
	require.NoError(t, err)
	require.NotEmpty(t, ourVdr)

	packager, err := New(afgoTestStorage, ourVdr.Registry())
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
