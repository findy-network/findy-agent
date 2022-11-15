package signature

import (
	"github.com/findy-network/findy-agent/core"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

type Signer struct {
	core.DID
}

func (s *Signer) Sign(src []byte) (dst []byte, err error) {
	defer err2.Returnf(&err, "signer DID")

	c := s.Packager().Crypto()
	kms := s.Packager().KMS()

	kh := try.To1(kms.Get(s.KID()))
	dst = try.To1(c.Sign(src, kh))

	return dst, nil
}

type Verifier struct {
	core.DID
}

func (v *Verifier) verify(verKey string, data, signature []byte) (err error) {
	defer err2.Returnf(&err, "verifier DID")

	keyBytes := try.To1(base58.Decode(verKey))
	keyHandle := try.To1(v.Packager().KMS().PubKeyBytesToHandle(keyBytes, kms.ED25519))
	try.To(v.Packager().Crypto().Verify(signature, data, keyHandle))

	return nil
}

func (v *Verifier) Verify(data, signature []byte) (err error) {
	return v.verify(v.VerKey(), data, signature)
}

func (v *Verifier) VerifyWithKey(key string, data, signature []byte) (err error) {
	return v.verify(key, data, signature)
}
