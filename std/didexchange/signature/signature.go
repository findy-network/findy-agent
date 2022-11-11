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

func (s *Verifier) Verify(data, signature []byte) (err error) {
	defer err2.Returnf(&err, "verifier DID")

	keyBytes := try.To1(base58.Decode(s.VerKey()))
	keyHandle := try.To1(s.Packager().KMS().PubKeyBytesToHandle(keyBytes, kms.ED25519))
	try.To(s.Packager().Crypto().Verify(signature, data, keyHandle))

	return nil
}
