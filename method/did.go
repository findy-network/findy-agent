package method

import (
	"strings"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/core"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func String(d string) string {
	s := strings.Split(d, ":")
	return s[1]
}

type Method int

const (
	MethodKey Method = 0 + iota
	MethodPeer
)

func New(hStorage managed.Wallet, method Method) (id core.DID, err error) {
	assert.D.True(method == MethodKey)
	return NewKey(hStorage)
}

type Key struct {
	kid string
	pk  []byte
	vkh any

	handle managed.Wallet
}

func NewKey(hStorage managed.Wallet) (id core.DID, err error) {
	defer err2.Annotate("new did:key", &err)

	keys := hStorage.Storage().KMS()
	kid, pk := try.To2(keys.CreateAndExportPubKeyBytes(kms.ED25519))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{handle: hStorage, kid: kid, pk: pk, vkh: kh}, nil
}

func NewKeyFromDID(
	hStorage managed.Wallet,
	didStr string,
) (
	id core.DID,
	err error,
) {
	defer err2.Annotate("new did:key from did", &err)

	keys := hStorage.Storage().KMS()
	pk := try.To1(fingerprint.PubKeyFromDIDKey(didStr))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{handle: hStorage, kid: "", pk: pk, vkh: kh}, nil
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

func (k Key) Storage() managed.Wallet {
	return k.handle
}

// TODO: this is mainly for indy but could be merged with SignKey?
func (k Key) VerKey() string {
	return string(k.pk)
}

func (k Key) Packager() api.Packager {
	return k.handle.Storage().OurPackager()
}
