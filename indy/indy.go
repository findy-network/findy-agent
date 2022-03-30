package indy

import (
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/spi/storage"
)

type Indy struct {
}

func (i *Indy) Open() error {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) Close() error {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) KMS() kms.KeyManager {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) DIDStorage() api.DIDStorage {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) ConnectionStorage() api.ConnectionStorage {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) CredentialStorage() api.CredentialStorage {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) OurPackager() api.Packager {
	panic("not implemented") // TODO: Implement
}

// We needed direct wrapping to because Go couldn't keep on with transitive
// type support of aggregated types.
func (i *Indy) OpenStore(name string) (storage.Store, error) {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) SetStoreConfig(name string, config storage.StoreConfiguration) error {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) GetStoreConfig(name string) (storage.StoreConfiguration, error) {
	panic("not implemented") // TODO: Implement
}

func (i *Indy) GetOpenStores() []storage.Store {
	panic("not implemented") // TODO: Implement
}
