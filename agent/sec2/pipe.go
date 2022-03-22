package sec2

import (
	"encoding/binary"
	"time"

	"github.com/findy-network/findy-agent/agent/packager"
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

// Pipe is secure way to transport data between DID connection. All agent to
// agent communication uses it. For its internal structure we must define the
// direction of the pipe.
type Pipe struct {
	In  core.DID
	Out core.DID

	mediaType string

	// TODO: move this to better place, to DID or storage? somewhere?
	Pckr *packager.Packager
}

// Verify verifies signature of the message and returns the verification key.
// Note! It throws err2 type of an error and needs an error handler in the call
// stack.
func (p Pipe) Verify(msg, signature []byte) (yes bool, vk string) {
	defer err2.Catch(func(err error) {
		glog.Error("error:", err)
		// want to be explicit, because underlaying API uses errors wrongly
		// for return non error information
		yes = false
	})
	c := p.Pckr.Crypto()

	try.To(c.Verify(signature, msg, p.Out.SignKey()))

	return true, ""
}

// Sign sings the message and returns the verification key. Note! It throws err2
// type of an error and needs an error handler in the call stack.
func (p Pipe) Sign(src []byte) (dst []byte, vk string) {
	defer err2.Catch(func(err error) {
		glog.Error("error:", err)
	})
	c := p.Pckr.Crypto()
	kms := p.Pckr.KMS()

	kh := try.To1(kms.Get(p.In.KID()))
	dst = try.To1(c.Sign(src, kh))

	return
}

// SignAndStamp sings and stamps a message and returns the verification key.
// Note! It throws err2 type of an error and needs an error handler in the call
// stack.
func (p Pipe) SignAndStamp(src []byte) (data, dst []byte, vk string) {
	now := getEpochTime()

	data = make([]byte, 8+len(src))
	binary.BigEndian.PutUint64(data[0:], uint64(now))

	l := copy(data[8:], src)
	if l != len(src) {
		glog.Warning("WARNING, NOT all bytes copied")
	}

	sign, verKey := p.Sign(data)
	return data, sign, verKey
}

// Pack packs the byte slice and returns verification key as well.
func (p Pipe) Pack(src []byte) (dst []byte, vk string, err error) {
	defer err2.Annotate("sec pipe pack", &err)
	assert.D.True(p.Pckr != nil)

	media := p.defMediaType()

	// pack an non empty envelope using packer selected by mediaType - should pass
	dst = try.To1(p.Pckr.PackMessage(&transport.Envelope{
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
	assert.D.True(p.Pckr != nil)

	env := try.To1(p.Pckr.UnpackMessage(src))
	dst = env.Message

	return
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
