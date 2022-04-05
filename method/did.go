package method

import (
	"strings"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
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
	//MethodPeer
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

func (k Key) Did() string {
	return k.kid
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

func (k Key) StartEndp(storageH managed.Wallet, connectionID string) {
	// todo: check how this is implemented in ssi.DID.
	// It seems that it's simple, but unnesseary.
}

func (k Key) Store(mgdWallet, mgdStorage managed.Wallet) {
	// todo: check the implementation from ssi.DID
	// it seems that there is nothing to do, all is saved already.
}

func (k Key) SavePairwiseForDID(mStorage managed.Wallet, theirDID core.DID, pw core.PairwiseMeta) {
	// todo: check ssi.DID, propably not needed
}

func (k Key) StoreResult() error {
	// todo: see ssi.DID
	return nil
}

func (k Key) AEndp() (ae service.Addr, err error) {
	assert.D.NoImplementation()
	return
}

func (k Key) SetAEndp(ae service.Addr) {
	assert.D.NoImplementation()
}

func (k Key) Route() []string {
	return []string{}
}

// TODO: this is mainly for indy but could be merged with SignKey?
func (k Key) VerKey() string {
	return string(k.pk)
}

func (k Key) Packager() api.Packager {
	return k.handle.Storage().OurPackager()
}
