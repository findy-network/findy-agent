package method

import (
	"github.com/findy-network/findy-agent/core"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type Method int

const (
	MethodKey Method = 0 + iota
	MethodPeer
)

func New(keys kms.KeyManager, method Method) (id core.DID, err error) {
	assert.D.True(method == MethodKey)
	return NewKey(keys)
}

type Key struct {
	kid string
	pk  []byte
	vkh any
}

func NewKey(keys kms.KeyManager) (id core.DID, err error) {
	defer err2.Annotate("new did:key", &err)

	kid, pk := try.To2(keys.CreateAndExportPubKeyBytes(kms.ED25519))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{kid: kid, pk: pk, vkh: kh}, nil
}

func NewKeyFromDID(keys kms.KeyManager, didStr string) (id core.DID, err error) {
	defer err2.Annotate("new did:key from did", &err)

	pk := try.To1(fingerprint.PubKeyFromDIDKey(didStr))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{kid: "", pk: pk, vkh: kh}, nil
}

func (k Key) String() string {
	// TODO: lazy fetch or move to constructor
	didkey, _ := fingerprint.CreateDIDKey(k.pk)

	return didkey
}

func (k Key) KID() string {
	return k.kid
}

func (k Key) SignKey() any {
	return k.vkh
}
