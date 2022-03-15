package packager

import (
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	cryptoapi "github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/crypto/tinkcrypto"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/packager"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/packer"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/packer/anoncrypt"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/packer/authcrypt"
	legacy "github.com/hyperledger/aries-framework-go/pkg/didcomm/packer/legacy/authcrypt"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/doc/jose"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/spi/storage"
	"github.com/lainio/err2"
)

type Packager struct {
	packager *packager.Packager
	storage  *mgddb.Storage
	registry vdr.Registry
	packers  []packer.Packer
	crypto   cryptoapi.Crypto
}

func New(
	agentStorage *mgddb.Storage,
	registry vdr.Registry,

) (p *Packager, err error) {
	defer err2.Annotate("packager new", &err)

	p = &Packager{
		storage:  agentStorage,
		registry: registry,
	}

	crypto, err := tinkcrypto.New()
	err2.Check(err)
	p.crypto = crypto

	// legacy authcrypt
	p.packers = append(p.packers, legacy.New(p))

	// authcrypt
	authPacker, err := authcrypt.New(p, jose.A256CBCHS512)
	err2.Check(err)
	p.packers = append(p.packers, authPacker)

	// anoncrypt
	anonPacker, err := anoncrypt.New(p, jose.A256GCM)
	err2.Check(err)
	p.packers = append(p.packers, anonPacker)

	pckr, err := packager.New(p)
	err2.Check(err)
	p.packager = pckr

	return p, err
}

func (p *Packager) PackMessage(messageEnvelope *transport.Envelope) ([]byte, error) {
	return p.packager.PackMessage(messageEnvelope)
}

func (p *Packager) UnpackMessage(encMessage []byte) (*transport.Envelope, error) {
	return p.packager.UnpackMessage(encMessage)
}

func (p *Packager) Packers() []packer.Packer {
	return p.packers
}

func (p *Packager) PrimaryPacker() packer.Packer {
	return p.packers[0]
}

func (p *Packager) VDRegistry() vdr.Registry {
	return p.registry
}

func (p *Packager) KMS() kms.KeyManager {
	return p.storage.KMS()
}

func (p *Packager) Crypto() cryptoapi.Crypto {
	return p.crypto
}

func (p *Packager) StorageProvider() storage.Provider {
	return p.storage
}
