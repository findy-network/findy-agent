package method

import (
	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/core"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type Key struct {
	Base
}

func NewKey(
	hStorage managed.Wallet,
	_ ...string,
) (
	id core.DID,
	err error,
) {
	defer err2.Handle(&err, "new did:key")

	keys := hStorage.Storage().KMS()
	kid, pk := try.To2(keys.CreateAndExportPubKeyBytes(kms.ED25519))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{Base{handle: hStorage, kid: kid, pk: pk, vkh: kh}}, nil
}

// NewKeyFromDID doesn't create a totally new did:key but it stores its pubkey
// to our KMS. We need it there for cryptos to work.
func NewKeyFromDID(
	hStorage managed.Wallet,
	didStr string,
) (
	id core.DID,
	err error,
) {
	defer err2.Handle(&err, "new did:key from did")

	keys := hStorage.Storage().KMS()
	pk := try.To1(fingerprint.PubKeyFromDIDKey(didStr))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{Base{handle: hStorage, kid: "", pk: pk, vkh: kh}}, nil
}

func (k Key) Route() []string {
	return []string{k.String()}
}

func (k Key) RecipientKeys() []string {
	return []string{k.String()}
}
func (k Key) NewDoc(_ service.Addr) core.DIDDoc {
	assert.NotImplemented("did:key doesn't support service endpoints")
	return nil
}
