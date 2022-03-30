package method

import (
	"github.com/findy-network/findy-agent/agent/storage/api"
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

func New(as api.AgentStorage, method Method) (id core.DID, err error) {
	assert.D.True(method == MethodKey)
	return NewKey(as)
}

type Key struct {
	kid string
	pk  []byte
	vkh any

	storage api.AgentStorage // TODO: this MUST be a managed handle!
}

func NewKey(as api.AgentStorage) (id core.DID, err error) {
	defer err2.Annotate("new did:key", &err)

	keys := as.KMS() // TODO: as must be a handle!
	kid, pk := try.To2(keys.CreateAndExportPubKeyBytes(kms.ED25519))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{storage: as, kid: kid, pk: pk, vkh: kh}, nil
}

func NewKeyFromDID(as api.AgentStorage, didStr string) (id core.DID, err error) {
	defer err2.Annotate("new did:key from did", &err)

	keys := as.KMS() // TODO: as must be a handle!
	pk := try.To1(fingerprint.PubKeyFromDIDKey(didStr))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{storage: as, kid: "", pk: pk, vkh: kh}, nil
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

func (k Key) Storage() api.AgentStorage {
	return k.storage // TODO: as must be a handle!
}

// TODO: this is mainly for indy but could be merged with SignKey?
func (k Key) VerKey() string {
	return string(k.pk)
}
