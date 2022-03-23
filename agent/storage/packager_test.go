package storage

import (
	"testing"

	"github.com/findy-network/findy-agent/agent/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/stretchr/testify/require"
)

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
