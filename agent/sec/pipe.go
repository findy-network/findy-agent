package sec

import (
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/indy"
	"github.com/findy-network/findy-agent/method"
	"github.com/golang/glog"
	cryptoapi "github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

var (
	defaultMediaType string
)

func Init(defMediaType string) {
	defaultMediaType = defMediaType
}

// Pipe is a secure way to transport data between DID connection. All agent to
// agent communication uses it. For its internal structure we must define the
// direction of the pipe.
type Pipe struct {
	In  core.DID
	Out core.DID
}

// NewPipeByVerkey creates a new secure pipe by our DID and other end's public
// key.
func NewPipeByVerkey(did core.DID, verkey string, route []string) *Pipe {
	assert.That(method.Accept(did, method.TypeSov))

	out := ssi.NewOutDid(verkey, route)

	return &Pipe{
		In:  did,
		Out: out,
	}
}

// Verify verifies signature of the message and returns the verification key.
// Note! It throws err2 type of error and needs an error handler in the call
// stack.
func (p Pipe) Verify(msg, signature []byte) (yes bool, vk string, err error) {
	defer err2.Returnf(&err, "pipe sign")

	c := p.crypto()
	try.To(c.Verify(signature, msg, p.Out.SignKey()))

	return true, p.Out.VerKey(), nil
}

// Sign sings the message and returns the verification key. Note! It throws err2
// type of error and needs an error handler in the call stack.
func (p Pipe) Sign(src []byte) (dst []byte, vk string, err error) {
	defer err2.Returnf(&err, "pipe sign")

	c := p.crypto()
	kms := p.packager().KMS()

	kh := try.To1(kms.Get(p.In.KID()))
	dst = try.To1(c.Sign(src, kh))
	vk = p.In.VerKey()

	return
}

// Pack packs the byte slice and returns verification key as well.
func (p Pipe) Pack(src []byte) (dst []byte, vk string, err error) {
	defer err2.Returnf(&err, "sec pipe pack")

	media := p.defMediaType()
	glog.V(15).Infoln("---- wallet handle:", p.In.Storage().Handle())

	toKeys := p.Out.RecipientKeys()
	route := p.Out.Route()
	toKeys = append(toKeys, route...)

	// pack a non-empty envelope using packer selected by mediaType - should pass
	dst = try.To1(p.packager().PackMessage(&transport.Envelope{
		MediaTypeProfile: media,
		Message:          src,
		FromKey:          []byte(p.In.String()),
		ToKeys:           toKeys,
	}))

	return
}

// Unpack unpacks the source bytes and returns our verification key as well.
func (p Pipe) Unpack(src []byte) (dst []byte, vk string, err error) {
	defer err2.Returnf(&err, "sec pipe unpack")

	env := try.To1(p.packager().UnpackMessage(src))
	dst = env.Message

	return
}

// IsNull returns true if pipe is null.
func (p Pipe) IsNull() bool {
	return p.In == nil
}

// EA returns endpoint of the agent.
func (p Pipe) EA() (ae service.Addr, err error) {
	return p.Out.AEndp()
}

func (p Pipe) defMediaType() string {
	return defaultMediaType
}

func (p Pipe) crypto() cryptoapi.Crypto {
	if p.packager() == nil {
		glog.V(10).Infoln("-- using Indy crypto as a default")
		return new(indy.Crypto)
	}
	return p.packager().Crypto()
}

func (p Pipe) packager() api.Packager {
	if p.In == nil {
		return nil
	}
	assert.D.True(p.In.Storage() != nil)

	return p.In.Packager()
}
