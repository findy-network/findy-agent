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
	"crypto/md5"
	"encoding/hex"
	"errors"

	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-grpc/crypto"
	"github.com/findy-network/findy-grpc/crypto/db"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

const emailB = "email_bucket"
const didB = "did_bucket"
const masterSecretB = "master_secret_bucket"

const emailBucket = 0
const didBucket = 1
const masterSecretBucket = 2

// ErrNotExists is an error for key not exist in the enclave.
var ErrNotExists = errors.New("key not exists")

var (
	sealedBoxFilename string

	buckets = [][]byte{
		[]byte(emailB),
		[]byte(didB),
		[]byte(masterSecretB),
	}

	theCipher *crypto.Cipher
)

// InitSealedBox initialize enclave's sealed box. This must be called once
// during the app life cycle.
func InitSealedBox(filename, backupName, key string) (err error) {
	if key != "" {
		glog.V(1).Info("init enclave with the key", filename)
		k, _ := hex.DecodeString(key)
		theCipher = crypto.NewCipher(k)
	} else {
		glog.Warningln("init enclave WITHOUT a key", filename)
	}

	sealedBoxFilename = filename
	if backupName == "" {
		backupName = "backup-" + sealedBoxFilename
	}
	return db.Init(db.Cfg{
		Filename:   sealedBoxFilename,
		BackupName: backupName,
		Buckets:    buckets,
	})
}

// Backup backups the enclave.
func Backup() {
	if err := db.Backup(); err != nil {
		glog.Errorln("enclave backup error:", err)
	}
}

// Close closes the enclave database
func Close() {
	err := db.Close()
	if err != nil {
		glog.Error(err)
	}
}

// WipeSealedBox closes and destroys the enclave permanently. This version only
// removes the sealed box file. In the future we might add sector wiping
// functionality.
func WipeSealedBox() {
	err := db.Wipe()
	if err != nil {
		glog.Error(err.Error())
	}
}

// NewWalletKey creates and stores a new indy wallet key to the enclave.
func NewWalletKey(email string) (key string, err error) {
	defer err2.Return(&err)

	value := &db.Data{Write: decrypt}
	already, err := db.GetKeyValueFromBucket(buckets[emailBucket],
		&db.Data{
			Data: []byte(email),
			Read: hash,
		},
		value)
	if already {
		return "", errors.New("key already exists")
	}

	key = err2.String.Try(generateKey())

	err2.Check(db.AddKeyValueToBucket(buckets[emailBucket],
		&db.Data{
			Data: []byte(key),
			Read: encrypt,
		},
		&db.Data{
			Data: []byte(email),
			Read: hash,
		},
	))

	return key, nil
}

func NewWalletMasterSecret(did string) (sec string, err error) {
	defer err2.Return(&err)

	value := &db.Data{Write: decrypt}
	already, err := db.GetKeyValueFromBucket(buckets[masterSecretBucket],
		&db.Data{
			Data: []byte(did),
			Read: hash,
		},
		value)
	if already {
		return "", errors.New("master secret already exists")
	}

	sec = utils.UUID()

	err2.Check(db.AddKeyValueToBucket(buckets[masterSecretBucket],
		&db.Data{
			Data: []byte(sec),
			Read: encrypt,
		},
		&db.Data{
			Data: []byte(did),
			Read: hash,
		},
	))

	return sec, nil
}

// WalletKeyNotExists returns true if a wallet key is not in the enclave
// associated by an email.
func WalletKeyNotExists(email string) bool {
	k, err := WalletKeyByEmail(email)
	return err == ErrNotExists && k == ""
}

// WalletKeyExists returns true if a wallet key is the enclave
// associated by an email.
func WalletKeyExists(email string) bool {
	return !WalletKeyNotExists(email)
}

// WalletKeyByEmail retrieves a wallet key from sealed box by an email
// associated to it.
func WalletKeyByEmail(email string) (key string, err error) {
	value := &db.Data{Write: decrypt}
	found := err2.Bool.Try(db.GetKeyValueFromBucket(buckets[emailBucket],
		&db.Data{
			Data: []byte(email),
			Read: hash,
		},
		value))
	if !found {
		return "", ErrNotExists
	}
	return string(value.Data), nil
}

// WalletKeyByDID retrieves a wallet key by a DID.
func WalletKeyByDID(DID string) (key string, err error) {
	value := &db.Data{Write: decrypt}
	found := err2.Bool.Try(db.GetKeyValueFromBucket(buckets[didBucket],
		&db.Data{
			Data: []byte(DID),
			Read: hash,
		},
		value))
	if !found {
		return "", ErrNotExists
	}
	return string(value.Data), nil
}

// WalletMasterSecretByDID retrieves a wallet master secret key by a DID.
func WalletMasterSecretByDID(DID string) (key string, err error) {
	value := &db.Data{Write: decrypt}
	found := err2.Bool.Try(db.GetKeyValueFromBucket(buckets[masterSecretBucket],
		&db.Data{
			Data: []byte(DID),
			Read: hash,
		},
		value))
	if !found {
		return "", ErrNotExists
	}
	return string(value.Data), nil
}

// SetKeysDID is a function to store a wallet key by its DID. We can retrieve a
// wallet key its DID with WalletKeyByDID.
func SetKeysDID(key, DID string) (err error) {
	return db.AddKeyValueToBucket(buckets[didBucket],
		&db.Data{
			Data: []byte(key),
			Read: encrypt,
		},
		&db.Data{
			Data: []byte(DID),
			Read: hash,
		},
	)
}

func generateKey() (key string, err error) {
	defer err2.Return(&err)

	r := <-wallet.GenerateKey("")
	err2.Check(r.Err())
	key = r.Str1()
	return key, nil
}

// all of the following has same signature. They also panic on error

// hash makes the cryptographic hash of the map key value. This prevents us to
// store key value index (email, DID) to the DB aka sealed box as plain text.
// Please use salt when implementing this.
func hash(key []byte) (k []byte) {
	if theCipher != nil {
		h := md5.Sum(key)
		return h[:]
	}
	return append(key[:0:0], key...)
}

// encrypt encrypts the actual wallet key value. This is used when data is
// stored do the DB aka sealed box.
func encrypt(value []byte) (k []byte) {
	if theCipher != nil {
		return theCipher.TryEncrypt(value)
	}
	return append(value[:0:0], value...)
}

// decrypt decrypts the actual wallet key value. This is used when data is
// retrieved from the DB aka sealed box.
func decrypt(value []byte) (k []byte) {
	if theCipher != nil {
		return theCipher.TryDecrypt(value)
	}
	return append(value[:0:0], value...)
}

// noop function if need e.g. tests
func _(value []byte) (k []byte) {
	println("noop called!")
	return value
}
