package api

import (
	cryptoapi "github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/spi/storage"
)

type AgentStorageConfig struct {
	AgentKey string
	AgentID  string
	FilePath string
}

type AgentStorage interface {
	Open() error
	Close() error

	KMS() kms.KeyManager

	DIDStorage() DIDStorage
	ConnectionStorage() ConnectionStorage
	CredentialStorage() CredentialStorage

	OurPackager() Packager

	OpenStore(name string) (storage.Store, error)
	SetStoreConfig(name string, config storage.StoreConfiguration) error
	GetStoreConfig(name string) (storage.StoreConfiguration, error)
	GetOpenStores() []storage.Store
}

type DIDStorage interface {
	SaveDID(did DID) error
	GetDID(id string) (*DID, error)
}

type Connection struct {
	ID            string
	MyDID         string
	TheirDID      string
	TheirEndpoint string
	TheirRoute    []string
}

type ConnectionStorage interface {
	SaveConnection(conn Connection) error
	GetConnection(id string) (*Connection, error)
	ListConnections() ([]Connection, error)
}

type CredentialStorage interface {
	// Similar than DIDStorage, relevant functionality for storing credential data
}

type Packager interface {
	KMS() kms.KeyManager
	Crypto() cryptoapi.Crypto
	//VDRegistry() vdr.Registry
	UnpackMessage(encMessage []byte) (*transport.Envelope, error)
	PackMessage(messageEnvelope *transport.Envelope) ([]byte, error)
	StorageProvider() storage.Provider
}
