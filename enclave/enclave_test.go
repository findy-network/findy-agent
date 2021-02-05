package enclave

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const dbFilename = "enclave.bolt"

const emailAddress = "test@email.com"
const emailNotCreated = "not@exists.email"

// key must be set from production environment, SHA-256, 32 bytes
const hexKey = "15308490f1e4026284594dd08d31291bc8ef2aeac730d0daf6ff87bb92d4336c"

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	_ = os.RemoveAll(dbFilename)
	_ = InitSealedBox(dbFilename, "", hexKey)
}

func tearDown() {
	Close()
	WipeSealedBox()
}

func TestNewWalletKey(t *testing.T) {
	k, err := NewWalletKey(emailAddress)
	assert.NoError(t, err)
	assert.NotEmpty(t, k)

	k2, err := WalletKeyByEmail(emailAddress)
	assert.NoError(t, err)
	assert.Equal(t, k, k2)

	k, err = NewWalletKey(emailAddress)
	assert.Error(t, err)
}

func TestSetKeysDID(t *testing.T) {
	const emailAddress = "test2@email.com"

	k, err := NewWalletKey(emailAddress)
	assert.NoError(t, err)
	assert.NotEmpty(t, k)
	key := k

	err = SetKeysDID(k, "TESTDID")
	assert.NoError(t, err)

	k, err = WalletKeyByDID("TESTDID")
	assert.NoError(t, err)
	assert.Equal(t, key, k)
}

func TestWalletKeyByEmail(t *testing.T) {
	key, err := WalletKeyByEmail(emailAddress)
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	key, err = WalletKeyByEmail(emailNotCreated)
	assert.Equal(t, ErrNotExists, err)
	assert.Empty(t, key)
	if ErrNotExists != err {
		t.Errorf("Not right (%s) error (%s)", ErrNotExists, err)
	}
}

func TestWalletKeyExists(t *testing.T) {
	notExists := WalletKeyNotExists(emailNotCreated)
	assert.True(t, notExists, "wallet not created")

	notExists = WalletKeyNotExists(emailAddress)
	assert.False(t, notExists, "wallet already created")
}

func TestNewWalletMasterSecret(t *testing.T) {
	sec, err := NewWalletMasterSecret("test_did")
	assert.NoError(t, err)
	assert.NotEmpty(t, sec)

	sec2, err := WalletMasterSecretByDID("test_did")
	assert.NoError(t, err)
	assert.NotEmpty(t, sec2)
	assert.Equal(t, sec, sec2)

	sec3, err := WalletMasterSecretByDID("wrong_test_did")
	assert.Error(t, err)
	assert.Empty(t, sec3)

}
