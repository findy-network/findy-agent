package enclave

import (
	"os"
	"testing"

	"github.com/lainio/err2/assert"
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
	assert.PushTester(t)
	defer assert.PopTester()

	k, err := NewWalletKey(emailAddress)
	assert.NoError(err)
	assert.NotEmpty(k)

	k2, err := WalletKeyByEmail(emailAddress)
	assert.NoError(err)
	assert.Equal(k, k2)

	_, err = NewWalletKey(emailAddress)
	assert.Error(err)
}

func TestSetKeysDID(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	const emailAddress = "test2@email.com"

	k, err := NewWalletKey(emailAddress)
	assert.NoError(err)
	assert.NotEmpty(k)
	key := k

	err = SetKeysDID(k, "TESTDID")
	assert.NoError(err)

	k, err = WalletKeyByDID("TESTDID")
	assert.NoError(err)
	assert.Equal(key, k)
}

func TestWalletKeyByEmail(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	key, err := WalletKeyByEmail(emailAddress)
	assert.NoError(err)
	assert.NotEmpty(key)

	key, err = WalletKeyByEmail(emailNotCreated)
	assert.That(ErrNotExists == err)
	assert.Empty(key)
	if ErrNotExists != err {
		t.Errorf("Not right (%s) error (%s)", ErrNotExists, err)
	}
}

func TestWalletKeyExists(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	notExists := WalletKeyNotExists(emailNotCreated)
	assert.That(notExists, "wallet not created")

	notExists = WalletKeyNotExists(emailAddress)
	assert.ThatNot(notExists, "wallet already created")
}

func TestNewWalletMasterSecret(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	sec, err := NewWalletMasterSecret("test_did")
	assert.NoError(err)
	assert.NotEmpty(sec)

	sec2, err := WalletMasterSecretByDID("test_did")
	assert.NoError(err)
	assert.NotEmpty(sec2)
	assert.Equal(sec, sec2)

	sec3, err := WalletMasterSecretByDID("wrong_test_did")
	assert.Error(err)
	assert.Empty(sec3)

}
