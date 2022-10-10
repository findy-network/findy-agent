package mgddb

import (
	"github.com/findy-network/findy-agent/agent/storage/api"
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
	"github.com/lainio/err2/try"
)

type Packager struct {
	packager *packager.Packager
	storage  *Storage
	registry vdr.Registry
	packers  []packer.Packer
	crypto   cryptoapi.Crypto
}

func NewPackagerFromStorage(
	agentStorage api.AgentStorage,
	registry vdr.Registry,
) (
	p *Packager,
	err error,
) {
	return NewPackager(agentStorage.(*Storage), registry)
}

func NewPackager(
	agentStorage *Storage,
	registry vdr.Registry,
) (
	p *Packager,
	err error,
) {
	defer err2.Returnf(&err, "packager new")

	p = &Packager{
		storage:  agentStorage,
		registry: registry,
	}

	crypto := try.To1(tinkcrypto.New())
	p.crypto = crypto

	// legacy authcrypt
	p.packers = append(p.packers, legacy.New(p))

	// authcrypt
	authPacker := try.To1(authcrypt.New(p, jose.A256CBCHS512))
	p.packers = append(p.packers, authPacker)

	// anoncrypt
	anonPacker := try.To1(anoncrypt.New(p, jose.A256GCM))
	p.packers = append(p.packers, anonPacker)

	pckr := try.To1(packager.New(p))
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
