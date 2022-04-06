package indy

import (
	"strings"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/spi/storage"
)

const MethodPrefix = "did:sov:"

func DID2KID(did string) string {
	return strings.TrimPrefix(did, MethodPrefix)
}

func New(handle int) *Indy {
	s := &Indy{Handle: handle, packager: nil}
	p := &Packager{storage: s}
	s.packager = p
	k := NewKMS(s)
	s.kms = k
	return s
}

type Indy struct {
	Handle int

	packager api.Packager
	kms      kms.KeyManager
}

func (i *Indy) Open() error {
	return nil
}

func (i *Indy) Close() error {
	return nil
}

func (i *Indy) KMS() kms.KeyManager {
	return i.kms
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
	return i.packager
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
