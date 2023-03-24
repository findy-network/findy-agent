package indy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/findy-network/findy-wrapper-go"
	indycrypto "github.com/findy-network/findy-wrapper-go/crypto"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdr"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/spi/storage"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

var (
	mediaProfile = transport.MediaTypeProfileDIDCommAIP1
)

type Handle struct {
	Wallet int
	VerKey string
}

type Packager struct {
	storage api.AgentStorage

	crypto Crypto
}

func (p *Packager) handle() int {
	return p.storage.(*Indy).Handle
}

func (p *Packager) KMS() kms.KeyManager {
	return p.storage.KMS()
}

func (p *Packager) Crypto() crypto.Crypto {
	return &p.crypto
}

func (p *Packager) VDRegistry() vdr.Registry {
	panic("not implemented") // TODO: Implement
}

func (p *Packager) UnpackMessage(
	encMessage []byte,
) (
	e *transport.Envelope,
	err error,
) {
	defer err2.Handle(&err, "indy unpack message")

	wallet := p.handle()

	if glog.V(5) {
		glog.Infof("<== Unpack: w(%d)\n", wallet)
	}

	r := <-indycrypto.UnpackMessage(wallet, encMessage)
	try.To(r.Err())

	unpacked := indycrypto.NewUnpacked(r.Bytes())

	// TODO: do not log sensitive data in production
	if glog.V(6) {
		glog.Infof("<== Unpacked: %s\n", unpacked.Message)
	}

	return &transport.Envelope{
		MediaTypeProfile: mediaProfile,
		Message:          []byte(unpacked.Message),
		FromKey:          []byte(unpacked.SenderVerkey),
		ToKey:            []byte(unpacked.RecipientVerkey),
	}, nil
}

func (p *Packager) PackMessage(envelope *transport.Envelope) (b []byte, err error) {
	defer err2.Handle(&err, "indy pack message")

	wallet := p.handle()
	toDID := envelope.ToKeys[0]
	assert.D.True(toDID != "")

	toVerKey := p.didStrToVerKey(toDID)
	senderKey := p.didStrToVerKey(string(envelope.FromKey))

	if glog.V(5) {
		glog.Infof("<== Pack: %s, %s", envelope.FromKey, senderKey)
		glog.Infof("<== Pack: w(%d) %s, %s", wallet,
			toDID, toVerKey)

		// TODO: do not log sensitive data in production
		if glog.V(6) {
			glog.Infof("<== Pack data: %s\n", string(envelope.Message))
		}
	}

	r := <-indycrypto.Pack(wallet, senderKey, envelope.Message, toVerKey)
	try.To(r.Err())

	res := r.Bytes()

	for i, toKey := range envelope.ToKeys {
		if i == 0 {
			continue
		}
		rKey := p.didStrToVerKey(toKey)
		glog.V(3).Infof("Packing with route key %s->%s",
			rKey, toKey)

		msgType := pltype.RoutingForward
		data := make(map[string]interface{})
		try.To(json.Unmarshal(res, &data))
		msg := aries.MsgCreator.Create(didcomm.MsgInit{
			Type: msgType,
			To:   toVerKey,
			Msg:  data,
		})
		fwdMsg := msg.FieldObj().(*common.Forward)

		// use anon-crypt for routing
		r := <-indycrypto.Pack(wallet, findy.NullString,
			dto.ToJSONBytes(fwdMsg), rKey)
		try.To(r.Err())

		res = r.Bytes()
		toVerKey = rKey
	}

	return res, nil
}

const SovVerKeyLen = 32

func (p *Packager) didStrToVerKey(vk string) string {
	vk = strings.TrimPrefix(vk, MethodPrefix)
	if len(vk) >= SovVerKeyLen {
		return vk
	}
	keyManager := p.KMS()
	vk = try.To1(keyManager.Get(vk)).(*Handle).VerKey
	return vk
}

func (p *Packager) StorageProvider() storage.Provider {
	panic("not implemented") // TODO: Implement
}

var (
	ErrWrongSignature = fmt.Errorf("signature validation failed")
)

type Crypto struct {
}

// Encrypt will encrypt msg and aad using a matching AEAD primitive in kh key handle of a public key
// returns:
//
//	cipherText in []byte
//	nonce in []byte
//	error in case of errors during encryption
func (c *Crypto) Encrypt(_ []byte, _ []byte, _ interface{}) ([]byte, []byte, error) {
	panic("not implemented") // TODO: Implement
}

// Decrypt will decrypt cipher with aad and given nonce using a matching AEAD primitive in kh key handle of a
// private key
// returns:
//
//	plainText in []byte
//	error in case of errors
func (c *Crypto) Decrypt(_ []byte, _ []byte, _ []byte, _ interface{}) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

// Sign will sign msg using a matching signature primitive in kh key handle of a private key
// returns:
//
//	signature in []byte
//	error in case of errors
func (c *Crypto) Sign(msg []byte, kh interface{}) (s []byte, err error) {
	defer err2.Handle(&err, "indy packager sign")

	handle := kh.(*Handle)
	r := <-indycrypto.SignMsg(handle.Wallet, handle.VerKey, msg)
	try.To(r.Err())
	return r.Bytes(), nil
}

// Verify will verify a signature for the given msg using a matching signature primitive in kh key handle of
// a public key
// returns:
//
//	error in case of errors or nil if signature verification was successful
func (c *Crypto) Verify(signature []byte, msg []byte, kh interface{}) (err error) {
	defer err2.Handle(&err, "indy packager verify")

	handle := kh.(*Handle)
	r := <-indycrypto.VerifySignature(handle.VerKey, msg, signature)
	try.To(r.Err())

	if !r.Yes() {
		return ErrWrongSignature
	}
	return nil
}

// ComputeMAC computes message authentication code (MAC) for code data
// using a matching MAC primitive in kh key handle
func (c *Crypto) ComputeMAC(_ []byte, _ interface{}) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

// VerifyMAC determines if mac is a correct authentication code (MAC) for data
// using a matching MAC primitive in kh key handle and returns nil if so, otherwise it returns an error.
func (c *Crypto) VerifyMAC(_ []byte, _ []byte, _ interface{}) error {
	panic("not implemented") // TODO: Implement
}

// WrapKey will execute key wrapping of cek using apu, apv and recipient public key 'recPubKey'.
// 'opts' allows setting the optional sender key handle using WithSender() option and the an authentication tag
// using WithTag() option. These allow ECDH-1PU key unwrapping (aka Authcrypt).
// The absence of these options uses ECDH-ES key wrapping (aka Anoncrypt). Another option that can
// be used is WithXC20PKW() to instruct the WrapKey to use XC20P key wrapping instead of the default A256GCM.
// returns:
//
//	RecipientWrappedKey containing the wrapped cek value
//	error in case of errors
func (c *Crypto) WrapKey(_ []byte, _ []byte, _ []byte, _ *crypto.PublicKey, _ ...crypto.WrapKeyOpts) (*crypto.RecipientWrappedKey, error) {
	panic("not implemented") // TODO: Implement
}

// UnwrapKey unwraps a key in recWK using recipient private key kh.
// 'opts' allows setting the optional sender key handle using WithSender() option and the an authentication tag
// using WithTag() option. These allow ECDH-1PU key unwrapping (aka Authcrypt).
// The absence of these options uses ECDH-ES key unwrapping (aka Anoncrypt). There is no need to
// use WithXC20PKW() for UnwrapKey since the function will use the wrapping algorithm based on recWK.Alg.
// returns:
//
//	unwrapped key in raw bytes
//	error in case of errors
func (c *Crypto) UnwrapKey(_ *crypto.RecipientWrappedKey, _ interface{}, _ ...crypto.WrapKeyOpts) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

// SignMulti will create a signature of messages using a matching signing primitive found in kh key handle of a
// private key.
// returns:
//
//	signature in []byte
//	error in case of errors
func (c *Crypto) SignMulti(_ [][]byte, _ interface{}) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

// VerifyMulti will verify a signature of messages using a matching signing primitive found in kh key handle of a
// public key.
// returns:
//
//	error in case of errors or nil if signature verification was successful
func (c *Crypto) VerifyMulti(_ [][]byte, _ []byte, _ interface{}) error {
	panic("not implemented") // TODO: Implement
}

// VerifyProof will verify a signature proof (generated e.g. by Verifier's DeriveProof() call) for revealedMessages
// using a matching signing primitive found in kh key handle of a public key.
// returns:
//
//	error in case of errors or nil if signature proof verification was successful
func (c *Crypto) VerifyProof(_ [][]byte, _ []byte, _ []byte, _ interface{}) error {
	panic("not implemented") // TODO: Implement
}

// DeriveProof will create a signature proof for a list of revealed messages using BBS signature (can be built using
// a Signer's SignMulti() call) and a matching signing primitive found in kh key handle of a public key.
// returns:
//
//	signature proof in []byte
//	error in case of errors
func (c *Crypto) DeriveProof(_ [][]byte, _ []byte, _ []byte, _ []int, _ interface{}) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}
