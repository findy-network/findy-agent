package mgddb

import (
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/kms/localkms"
	"github.com/hyperledger/aries-framework-go/pkg/secretlock"
	"github.com/hyperledger/aries-framework-go/pkg/secretlock/noop"
	"github.com/hyperledger/aries-framework-go/spi/storage"
	"github.com/lainio/err2"
)

type kmsStorage struct {
	kms      kms.KeyManager
	owner    *Storage
	noopLock secretlock.Service
}

func newKmsStorage(owner *Storage) (k *kmsStorage, err error) {
	defer err2.Annotate("new kms storage", &err)

	k = &kmsStorage{
		owner:    owner,
		noopLock: &noop.NoLock{},
	}

	localKms, err := localkms.New("local-lock://primary/test/", k) // TODO: figure out uri purpose
	err2.Check(err)

	k.kms = localKms

	return
}

func (k *kmsStorage) StorageProvider() storage.Provider {
	return k.owner
}

func (k *kmsStorage) SecretLock() secretlock.Service {
	return k.noopLock
}

func (k *kmsStorage) KMS() kms.KeyManager {
	return k.kms
}
