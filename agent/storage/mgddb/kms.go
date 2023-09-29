package mgddb

import (
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/kms/localkms"
	"github.com/hyperledger/aries-framework-go/pkg/secretlock"
	"github.com/hyperledger/aries-framework-go/pkg/secretlock/noop"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type kmsStorage struct {
	kms      kms.KeyManager
	owner    kms.Store
	noopLock secretlock.Service
}

func newKmsStorage(owner *Storage) (k *kmsStorage, err error) {
	defer err2.Handle(&err, "new kms storage")

	k = &kmsStorage{
		owner:    try.To1(kms.NewAriesProviderWrapper(owner)),
		noopLock: &noop.NoLock{},
	}

	localKms := try.To1(localkms.New("local-lock://primary/test/", k)) // TODO: figure out uri purpose

	k.kms = localKms

	return
}

func (k *kmsStorage) StorageProvider() kms.Store {
	return k.owner
}

func (k *kmsStorage) SecretLock() secretlock.Service {
	return k.noopLock
}

func (k *kmsStorage) KMS() kms.KeyManager {
	return k.kms
}
