package api

import (
	"github.com/hyperledger/aries-framework-go/pkg/kms"
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
}

type DIDStorage interface {
	AddDID(did DID) error
	GetDID(id string) (*DID, error)
}

type Connection struct {
	ID            string
	OurDID        string
	TheirDID      string
	TheirEndpoint string
	TheirRoute    []string
}

type ConnectionStorage interface {
	AddConnection(conn Connection) error
	GetConnection(id string) (*Connection, error)
	ListConnections() ([]Connection, error)
}

type CredentialStorage interface {
	// Similar than DIDStorage, relevant functionality for storing credential data
}
