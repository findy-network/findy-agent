package sec2

import (
	"encoding/binary"
	"time"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/core"
	"github.com/golang/glog"
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

	mediaType string
}

// Verify verifies signature of the message and returns the verification key.
// Note! It throws err2 type of error and needs an error handler in the call
// stack.
func (p Pipe) Verify(msg, signature []byte) (yes bool, vk string, err error) {
	defer err2.Annotate("pipe sign", &err)

	c := p.packager().Crypto()
	try.To(c.Verify(signature, msg, p.Out.SignKey()))

	return true, "", nil
}

// Sign sings the message and returns the verification key. Note! It throws err2
// type of error and needs an error handler in the call stack.
func (p Pipe) Sign(src []byte) (dst []byte, vk string, err error) {
	defer err2.Annotate("pipe sign", &err)

	c := p.packager().Crypto()
	kms := p.packager().KMS()

	kh := try.To1(kms.Get(p.In.KID()))
	dst = try.To1(c.Sign(src, kh))

	return
}

// SignAndStamp sings and stamps a message and returns the verification key.
// Note! It throws err2 type of error and needs an error handler in the call
// stack.
func (p Pipe) SignAndStamp(src []byte) (data, dst []byte, vk string, err error) {
	defer err2.Return(&err)

	now := getEpochTime()

	data = make([]byte, 8+len(src))
	binary.BigEndian.PutUint64(data[0:], uint64(now))

	l := copy(data[8:], src)
	if l != len(src) {
		glog.Warning("WARNING, NOT all bytes copied")
	}

	sign, verKey := try.To2(p.Sign(data))
	return data, sign, verKey, nil
}

// Pack packs the byte slice and returns verification key as well.
func (p Pipe) Pack(src []byte) (dst []byte, vk string, err error) {
	defer err2.Annotate("sec pipe pack", &err)

	media := p.defMediaType()

	// pack a non-empty envelope using packer selected by mediaType - should pass
	dst = try.To1(p.packager().PackMessage(&transport.Envelope{
		MediaTypeProfile: media,
		Message:          src,
		FromKey:          []byte(p.In.String()),
		ToKeys:           []string{p.Out.String()},
	}))

	return
}

// Unpack unpacks the source bytes and returns our verification key as well.
func (p Pipe) Unpack(src []byte) (dst []byte, vk string, err error) {
	defer err2.Annotate("sec pipe unpack", &err)

	env := try.To1(p.packager().UnpackMessage(src))
	dst = env.Message

	return
}

// IsNull returns true if pipe is null.
func (p Pipe) IsNull() bool {
	return p.In == nil
}

func getEpochTime() int64 {
	return time.Now().Unix()
}

func (p Pipe) defMediaType() string {
	if p.mediaType == "" {
		return defaultMediaType
	}
	return p.mediaType
}

func (p Pipe) packager() api.Packager {
	assert.D.True(p.In.Storage() != nil)

	// TODO: Storage() return a handle!
	return p.In.Storage().OurPackager()
}
