/*
Package enclave is a server-side Secure Enclave. It offers a secure and sealed
storage to store indy wallet keys on the Agency server.

Urgent! This version does not implement internal hash(), encrypt, and decrypt()
functions. We must implement these three functions before production. We will
offer implementations of them when the server-side crypto solution and the Key
Storage is selected. Possible candidates are AWS Nitro, etc. We also bring
addon/plugin system for cryptos when first implementation is done.
*/
package enclave

import (
	"errors"
	"os"

	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

const emailBucket = "email_bucket"
const didBucket = "did_bucket"
const masterSecretBucket = "master_secret_bucket"

var sealedBoxFilename string

// InitSealedBox initialize enclave's sealed box. This must be called once
// during the app life cycle.
func InitSealedBox(filename string) (err error) {
	glog.V(1).Info("init enclave", filename)
	sealedBoxFilename = filename
	return open(filename)
}

// WipeSealedBox closes and destroys the enclave permanently. This version only
// removes the sealed box file. In the future we might add sector wiping
// functionality.
func WipeSealedBox() {
	if db != nil {
		Close()
	}

	err := os.RemoveAll(sealedBoxFilename)
	if err != nil {
		println(err.Error())
	}
}

// NewWalletKey creates and stores a new indy wallet key to the enclave.
func NewWalletKey(email string) (key string, err error) {
	defer err2.Return(&err)

	key, err = decrypt(getKeyValueFromBucket(emailBucket, hash(email)))
	if key != "" {
		return "", errors.New("key already exists")
	}

	key = err2.String.Try(generateKey())
	err2.Check(addKeyValueToBucket(emailBucket, encrypt(key), hash(email)))

	return key, nil
}

func NewWalletMasterSecret(did string) (sec string, err error) {
	defer err2.Return(&err)

	sec, err = decrypt(getKeyValueFromBucket(masterSecretBucket, hash(did)))
	if sec != "" {
		return "", errors.New("master secret already exists")
	}

	sec = utils.UUID()
	err2.Check(addKeyValueToBucket(masterSecretBucket, encrypt(sec), hash(did)))

	return sec, nil
}

// WalletKeyNotExists returns true if a wallet key is not in the enclave
// associated by an email.
func WalletKeyNotExists(email string) bool {
	k, err := WalletKeyByEmail(hash(email))
	return err == ErrNotExists && k == ""
}

// WalletKeyByEmail retrieves a wallet key from sealed box by an email
// associated to it.
func WalletKeyByEmail(email string) (key string, err error) {
	return decrypt(getKeyValueFromBucket(emailBucket, hash(email)))
}

// WalletKeyByDID retrieves a wallet key by a DID.
func WalletKeyByDID(DID string) (key string, err error) {
	return decrypt(getKeyValueFromBucket(didBucket, hash(DID)))
}

// WalletMasterSecretByDID retrieves a wallet master secret key by a DID.
func WalletMasterSecretByDID(DID string) (key string, err error) {
	return decrypt(getKeyValueFromBucket(masterSecretBucket, hash(DID)))
}

// SetKeysDID is a function to store a wallet key by its DID. We can retrieve a
// wallet key its DID with WalletKeyByDID.
func SetKeysDID(key, DID string) (err error) {
	return addKeyValueToBucket(didBucket, encrypt(key), hash(DID))
}

func generateKey() (key string, err error) {
	defer err2.Return(&err)

	r := <-wallet.GenerateKey("")
	err2.Check(r.Err())
	key = r.Str1()
	return key, nil
}

// Todo: these dummy functions must be implemented before production.

// hash makes the cryptographic hash of the map key value. This prevents us to
// store key value index (email, DID) to the DB aka sealed box as plain text.
// Please use salt when implementing this.
func hash(mapKeyValue string) (k string) {
	return mapKeyValue
}

// encrypt encrypts the actual wallet key value. This is used when data is
// stored do the DB aka sealed box.
func encrypt(keyValue string) (k string) {
	return keyValue
}

// decrypt decrypts the actual wallet key value. This is used when data is
// retrieved from the DB aka sealed box.
func decrypt(keyValue string, e error) (k string, err error) {
	return keyValue, e
}
