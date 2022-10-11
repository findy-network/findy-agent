package storage

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/google/tink/go/keyset"
	"github.com/hyperledger/aries-framework-go/pkg/kms"

	"github.com/lainio/err2/assert"
)

func TestKMSCreateAndExportPubKeyBytes(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.KMS()
			keyID, keyBytes, err := store.CreateAndExportPubKeyBytes(kms.ED25519Type)
			assert.NoError(err)
			assert.NotEmpty(keyID)
			assert.SNotEmpty(keyBytes)
		})
	}
}

func TestKMSCreate(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.KMS()

			keyID, handle, err := store.Create(kms.ED25519Type)
			assert.NoError(err)
			assert.NotEmpty(keyID)
			assert.INotNil(handle)
		})
	}
}

func verifyKMSGet(t *testing.T, store kms.KeyManager, keyID string, keyHandle interface{}) {
	switch putKeyHandle := keyHandle.(type) {
	case *keyset.Handle:
		handlePutPrimitives, e := putKeyHandle.Primitives()
		assert.NoError(e)
		assert.INotNil(handlePutPrimitives)

		handleGet, err := store.Get(keyID)
		assert.NoError(err)
		assert.INotNil(handleGet)

		handleGetPrimitives, e := handleGet.(*keyset.Handle).Primitives()
		assert.NoError(e)
		assert.INotNil(handleGetPrimitives)

		assert.MNotEmpty(handlePutPrimitives.Entries)
		assert.Equal(len(handlePutPrimitives.Entries), len(handleGetPrimitives.Entries))
	default:
		t.Errorf("unexpected handle type: %v", keyHandle)
	}
}

func TestKMSGet(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.KMS()

			keyID, handlePut, err := store.Create(kms.ED25519Type)
			assert.NoError(err)
			assert.NotEmpty(keyID)
			assert.INotNil(handlePut)

			verifyKMSGet(t, store, keyID, handlePut)
		})
	}
}

func TestKMSGetAfterClose(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			keyID, handlePut, err := testCase.storage.KMS().Create(kms.ED25519Type)
			assert.NoError(err)
			assert.NotEmpty(keyID)
			assert.INotNil(handlePut)

			verifyKMSGet(t, testCase.storage.KMS(), keyID, handlePut)

			assert.NoError(testCase.storage.Close())

			err = testCase.storage.Open()
			assert.NoError(err)

			verifyKMSGet(t, testCase.storage.KMS(), keyID, handlePut)
		})
	}
}

func TestKMSExportPubKeyBytes(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.KMS()

			keyID, handlePut, err := store.Create(kms.ED25519Type)
			assert.NoError(err)
			assert.NotEmpty(keyID)
			assert.INotNil(handlePut)

			bytes, _, err := store.ExportPubKeyBytes(keyID)
			assert.NoError(err)
			assert.INotNil(bytes)
		})
	}
}

func TestKMSPubKeyBytesToHandle(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.KMS()
			keyID, handlePut, err := store.Create(kms.ED25519Type)
			assert.NoError(err)
			assert.NotEmpty(keyID)
			assert.INotNil(handlePut)

			bytes, _, err := store.ExportPubKeyBytes(keyID)
			assert.NoError(err)
			assert.SNotEmpty(bytes)

			handleGet, err := store.PubKeyBytesToHandle(bytes, kms.ED25519Type)
			assert.NoError(err)
			assert.INotNil(handleGet)

			handlePutPrimitives, e := handlePut.(*keyset.Handle).Primitives()
			assert.NoError(e)
			assert.INotNil(handlePutPrimitives)

			handleGetPrimitives, e := handleGet.(*keyset.Handle).Primitives()
			assert.NoError(e)
			assert.INotNil(handleGetPrimitives)

			assert.MNotEmpty(handlePutPrimitives.Entries)
			assert.Equal(len(handlePutPrimitives.Entries), len(handleGetPrimitives.Entries))
		})
	}
}

func TestKMSRotate(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.KMS()

			keyID, handlePut, err := store.Create(kms.ED25519Type)
			assert.NoError(err)
			assert.NotEmpty(keyID)
			assert.INotNil(handlePut)

			newKeyID, handleRotate, err := store.Rotate(kms.ED25519Type, keyID)
			assert.NoError(err)
			assert.NotEmpty(newKeyID)
			assert.INotNil(handleRotate)
			assert.NotEqual(keyID, newKeyID)

			handleGet, err := store.Get(newKeyID)
			assert.NoError(err)
			assert.INotNil(handleGet)
		})
	}
}

func TestKMSImportPrivateKey(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			store := testCase.storage.KMS()
			pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
			assert.NoError(err)

			ksID, _, err := store.ImportPrivateKey(privKey, kms.ED25519Type)
			assert.NoError(err)

			pubKeyBytes, _, err := store.ExportPubKeyBytes(ksID)
			assert.NoError(err)
			assert.DeepEqual(pubKey, ed25519.PublicKey(pubKeyBytes))
		})
	}
}
