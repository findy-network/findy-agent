package sec

import (
	"encoding/binary"
	"time"

	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-wrapper-go/crypto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

// Pipe is secure way to transport data between DID connection. All agent to
// agent communication uses it. For its internal structure we must define the
// direction of the pipe.
type Pipe struct {
	ssi.In
	ssi.Out
}

// NewPipeByVerkey creates a new secure pipe by our DID and other end's public
// key.
func NewPipeByVerkey(did *ssi.DID, verkey string) *Pipe {
	return &Pipe{
		In:  did,
		Out: ssi.NewDid("", verkey), // we know verkey only
	}
}

// Encrypt encrypts the byte slice. Note! It throws err2 type of an error and
// needs an error handler in the call stack.
func (p Pipe) Encrypt(src []byte) (dst []byte) {
	if glog.V(5) {
		glog.Info("<== Encrypt: ", p.Out.Did())
	}
	r := <-crypto.AnonCrypt(p.Out.VerKey(), src)
	err2.Check(r.Err())
	return r.Bytes()
}

// Decrypt decrypts the byte slice.
func (p Pipe) Decrypt(src []byte) (dst []byte) {
	dst, _ = p.DecryptGiveKey(src)
	return
}

// Verify verifies signature of the message and returns the verification key.
// Note! It throws err2 type of an error and needs an error handler in the call
// stack.
func (p Pipe) Verify(msg, signature []byte) (yes bool, vk string) {
	vk = p.Out.VerKey()

	r := <-crypto.VerifySignature(vk, msg, signature)
	err2.Check(r.Err())
	return r.Yes(), vk
}

// Sign sings the message and returns the verification key. Note! It throws err2
// type of an error and needs an error handler in the call stack.
func (p Pipe) Sign(src []byte) (dst []byte, vk string) {
	wallet := p.wallet()
	vk = p.In.VerKey()

	r := <-crypto.SignMsg(wallet, vk, src)
	err2.Check(r.Err())
	return r.Bytes(), vk
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

// DecryptGiveKey decrypts scr bytes and returns the verkey as well. Note! It
// throws an err2 exception. So, there should be at least one error handler in
// the call stack.
func (p Pipe) DecryptGiveKey(src []byte) (dst []byte, vk string) {
	wallet := p.wallet()

	if glog.V(5) {
		glog.Infof("==> Decrypt: %s, w(%d)\n", p.In.VerKey(), wallet)
	}
	vk = p.In.VerKey()
	r := <-crypto.AnonDecrypt(wallet, vk, src)
	err2.Check(r.Err())

	return r.Bytes(), vk
}

// Pack packs the byte slice and returns verification key as well.
func (p Pipe) Pack(src []byte) (dst []byte, vk string, err error) {
	wallet := p.wallet()
	vk = p.Out.VerKey()

	if vk == "" {
		glog.Error("verification key cannot be empty")
		panic("programming error")
	}

	if glog.V(5) {
		glog.Infof("<== Pack: %s/%s, w(%d)\n", p.Out.Did(), vk, wallet)

		// TODO: do not log sensitive data in production
		if glog.V(6) {
			glog.Infof("<== Pack data: %s\n", string(src))
		}
	}

	senderKey := p.In.VerKey()

	r := <-crypto.Pack(wallet, senderKey, src, vk)
	if r.Err() != nil {
		return nil, "", r.Err()
	}

	return r.Bytes(), vk, nil
}

// Unpack unpacks the source bytes and returns our verification key as well.
func (p Pipe) Unpack(src []byte) (dst []byte, vk string, err error) {
	wallet := p.wallet()
	vk = p.In.VerKey()

	if glog.V(5) {
		glog.Infof("<== Unpack: w(%d)\n", wallet)
	}

	r := <-crypto.UnpackMessage(wallet, src)
	if r.Err() != nil {
		return nil, "", r.Err()
	}

	res := crypto.NewUnpacked(r.Bytes()).Bytes()

	// TODO: do not log sensitive data in production
	if glog.V(6) {
		glog.Infof("<== Unpacked: %s\n", string(res))
	}

	return res, vk, nil
}

// IsNull returns true if pipe is null.
func (p Pipe) IsNull() bool {
	return p.In == nil
}

// EA returns endpoint of the agent.
func (p Pipe) EA() (ae service.Addr, err error) {
	return p.Out.AEndp()
}

func (p Pipe) wallet() int {
	wallet := p.In.Wallet()
	if wallet == 0 {
		panic("wallet not set")
	}
	return wallet
}

func getEpochTime() int64 {
	return time.Now().Unix()
}
