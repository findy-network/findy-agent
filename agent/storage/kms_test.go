package storage

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/google/tink/go/keyset"
	"github.com/hyperledger/aries-framework-go/pkg/kms"

	"github.com/stretchr/testify/require"
)

func TestKMSCreateAndExportPubKeyBytes(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.KMS()
			keyID, keyBytes, err := store.CreateAndExportPubKeyBytes(kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, keyID)
			require.NotEmpty(t, keyBytes)
		})
	}
}

func TestKMSCreate(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.KMS()

			keyID, handle, err := store.Create(kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, keyID)
			require.NotEmpty(t, handle)
		})
	}
}

func verifyKMSGet(t *testing.T, store kms.KeyManager, keyID string, keyHandle interface{}) {
	switch putKeyHandle := keyHandle.(type) {
	case *keyset.Handle:
		handlePutPrimitives, e := putKeyHandle.Primitives()
		require.NoError(t, e)
		require.NotEmpty(t, handlePutPrimitives)

		handleGet, err := store.Get(keyID)
		require.NoError(t, err)
		require.NotEmpty(t, handleGet)

		handleGetPrimitives, e := handleGet.(*keyset.Handle).Primitives()
		require.NoError(t, e)
		require.NotEmpty(t, handleGetPrimitives)

		require.Greater(t, len(handlePutPrimitives.Entries), 0)
		require.Equal(t, len(handlePutPrimitives.Entries), len(handleGetPrimitives.Entries))
	default:
		t.Errorf("unexpected handle type: %v", keyHandle)
	}
}

func TestKMSGet(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.KMS()

			keyID, handlePut, err := store.Create(kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, keyID)
			require.NotEmpty(t, handlePut)

			verifyKMSGet(t, store, keyID, handlePut)
		})
	}
}

func TestKMSGetAfterClose(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			keyID, handlePut, err := testCase.storage.KMS().Create(kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, keyID)
			require.NotEmpty(t, handlePut)

			verifyKMSGet(t, testCase.storage.KMS(), keyID, handlePut)

			require.NoError(t, testCase.storage.Close())

			err = testCase.storage.Open()
			require.NoError(t, err)

			verifyKMSGet(t, testCase.storage.KMS(), keyID, handlePut)
		})
	}
}

func TestKMSExportPubKeyBytes(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.KMS()

			keyID, handlePut, err := store.Create(kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, keyID)
			require.NotEmpty(t, handlePut)

			bytes, err := store.ExportPubKeyBytes(keyID)
			require.NoError(t, err)
			require.NotEmpty(t, bytes)
		})
	}
}

func TestKMSPubKeyBytesToHandle(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.KMS()
			keyID, handlePut, err := store.Create(kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, keyID)
			require.NotEmpty(t, handlePut)

			bytes, err := store.ExportPubKeyBytes(keyID)
			require.NoError(t, err)
			require.NotEmpty(t, bytes)

			handleGet, err := store.PubKeyBytesToHandle(bytes, kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, handleGet)

			handlePutPrimitives, e := handlePut.(*keyset.Handle).Primitives()
			require.NoError(t, e)
			require.NotEmpty(t, handlePutPrimitives)

			handleGetPrimitives, e := handleGet.(*keyset.Handle).Primitives()
			require.NoError(t, e)
			require.NotEmpty(t, handleGetPrimitives)

			require.Greater(t, len(handlePutPrimitives.Entries), 0)
			require.Equal(t, len(handlePutPrimitives.Entries), len(handleGetPrimitives.Entries))
		})
	}
}

func TestKMSRotate(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.KMS()

			keyID, handlePut, err := store.Create(kms.ED25519Type)
			require.NoError(t, err)
			require.NotEmpty(t, keyID)
			require.NotEmpty(t, handlePut)

			newKeyID, handleRotate, err := store.Rotate(kms.ED25519Type, keyID)
			require.NoError(t, err)
			require.NotEmpty(t, newKeyID)
			require.NotEmpty(t, handleRotate)
			require.NotEqual(t, keyID, newKeyID)

			handleGet, err := store.Get(newKeyID)
			require.NoError(t, err)
			require.NotEmpty(t, handleGet)
		})
	}
}

func TestKMSImportPrivateKey(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.KMS()
			pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
			require.NoError(t, err)

			ksID, _, err := store.ImportPrivateKey(privKey, kms.ED25519Type)
			require.NoError(t, err)

			pubKeyBytes, err := store.ExportPubKeyBytes(ksID)
			require.NoError(t, err)
			require.EqualValues(t, pubKey, pubKeyBytes)
		})
	}
}
