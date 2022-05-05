package method

import (
	"strings"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/core"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/peer"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

func String(d string) string {
	s := strings.Split(d, ":")
	return s[1]
}

func MethodType(s string) Type {
	t, ok := methodTypes[String(s)]
	if !ok {
		glog.Warningf("cannot compute did method from '%s'", s)
		return TypeUnknown
	}
	return t
}

func Accept(did core.DID, t Type) bool {
	return MethodType(did.String()) == t
}

var methodTypes = map[string]Type{
	"unknown": TypeUnknown,
	"key":     TypeKey,
	"peer":    TypePeer,
	"sov":     TypeSov,
	"indy":    TypeIndy,
}

type Type int

const (
	TypeUnknown Type = 0 + iota
	TypeKey
	TypePeer
	TypeSov
	TypeIndy
)

func (t Type) String() string {
	return []string{"unknown", "key", "peer", "sov", "indy"}[t]
}

func New(
	method Type,
	hStorage managed.Wallet,
	args ...string,
) (
	id core.DID,
	err error,
) {
	switch method {
	case TypePeer:
		return NewPeer(hStorage, args...)
	case TypeKey:
		return NewKey(hStorage, args...)
	default:
		assert.D.Truef(false, "did method (%v) not supported", method)
	}
	return
}

type Base struct {
	kid string
	pk  []byte
	vkh any

	doc *did.Doc

	handle managed.Wallet
}

type Peer struct {
	Base
}

type Key struct {
	Base
}

func NewKey(
	hStorage managed.Wallet,
	args ...string,
) (
	id core.DID,
	err error,
) {
	defer err2.Annotate("new did:key", &err)

	keys := hStorage.Storage().KMS()
	kid, pk := try.To2(keys.CreateAndExportPubKeyBytes(kms.ED25519))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{Base{handle: hStorage, kid: kid, pk: pk, vkh: kh}}, nil
}

func NewPeer(
	hStorage managed.Wallet,
	args ...string,
) (
	id core.DID,
	err error,
) {
	defer err2.Annotate("new did:peer", &err)

	keys := hStorage.Storage().KMS()
	kid, pk := try.To2(keys.CreateAndExportPubKeyBytes(kms.ED25519))
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	key := did.VerificationMethod{
		ID:         kid,
		Type:       "Ed25519VerificationKey2018",
		Controller: "",
		Value:      pk,
	}
	doc := try.To1(peer.NewDoc(
		[]did.VerificationMethod{key},
		did.WithAuthentication([]did.Verification{{
			VerificationMethod: key,
			Relationship:       0,
			Embedded:           true,
		}}),
		did.WithService([]did.Service{{
			ID:              "didcomm",
			Type:            "did-communication",
			Priority:        0,
			RecipientKeys:   []string{base58.Encode(pk)},
			ServiceEndpoint: args[0], // TODO: from where originally?
		}}),
	))

	return Peer{Base{handle: hStorage, kid: kid, pk: pk, vkh: kh, doc: doc}}, nil
}

func NewFromDID(
	hStorage managed.Wallet,
	didStr string,
) (
	id core.DID,
	err error,
) {
	switch MethodType(didStr) {
	case TypePeer:
		return NewPeerFromDID(hStorage, didStr)
	case TypeKey:
		return NewKeyFromDID(hStorage, didStr)
	default:
		assert.NotImplemented()
	}
	return
}

func NewPeerFromDID(
	hStorage managed.Wallet,
	didStr string,
) (
	id core.DID,
	err error,
) {
	defer err2.Annotate("new did:peer from did", &err)

	keys := hStorage.Storage().KMS()

	// TODO: get pubkey from peer did string
	//  - that cannot be done, the actual signing key has to get form diddoc

	pk := try.To1(fingerprint.PubKeyFromDIDKey(didStr))

	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Key{Base{handle: hStorage, kid: "", pk: pk, vkh: kh}}, nil
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

	return Key{Base{handle: hStorage, kid: "", pk: pk, vkh: kh}}, nil
}

// String returns URI formated DID
func (b Base) String() string {
	// TODO: lazy fetch or move to constructor
	didkey, _ := fingerprint.CreateDIDKey(b.pk)

	return didkey
}

func (b Base) Did() string {
	return b.kid
}

func (b Base) KID() string {
	return b.kid
}

func (b Base) SignKey() any {
	return b.vkh
}

func (b Base) Storage() managed.Wallet {
	return b.handle
}

func (b Base) StartEndp(storageH managed.Wallet, connectionID string) {
	// todo: check how this is implemented in ssi.DID.
	// It seems that it's simple, but unnesseary.
}

func (b Base) Store(mgdWallet, mgdStorage managed.Wallet) {
	// todo: check the implementation from ssi.DID
	// it seems that there is nothing to do, all is saved already.
}

func (b Base) SavePairwiseForDID(mStorage managed.Wallet, theirDID core.DID, pw core.PairwiseMeta) {
	// todo: check ssi.DID, propably not needed
}

func (b Base) StoreResult() error {
	// todo: see ssi.DID
	return nil
}

func (b Base) AEndp() (ae service.Addr, err error) {
	assert.D.NoImplementation()
	return
}

func (b Base) SetAEndp(ae service.Addr) {
	assert.D.NoImplementation()
}

func (b Base) Route() []string {
	return []string{}
}

// TODO: this is mainly for indy but could be merged with SignKey?
func (b Base) VerKey() string {
	return string(b.pk)
}

func (b Base) Packager() api.Packager {
	return b.handle.Storage().OurPackager()
}

func (b Base) DOC() core.DIDDoc {
	return b.doc
}

func (k Key) NewDoc(ae service.Addr) core.DIDDoc {
	assert.NotImplemented("did:key doesn't support service endpoints")
	return nil
}

func (p Peer) NewDoc(ae service.Addr) core.DIDDoc {
	key := did.VerificationMethod{
		ID:         p.kid,
		Type:       "Ed25519VerificationKey2018",
		Controller: "",
		Value:      p.pk,
	}
	doc := try.To1(peer.NewDoc(
		[]did.VerificationMethod{key},
		did.WithAuthentication([]did.Verification{{
			VerificationMethod: key,
			Relationship:       0,
			Embedded:           true,
		}}),
		did.WithService([]did.Service{{
			ID:              "didcomm",
			Type:            "did-communication",
			Priority:        0,
			RecipientKeys:   []string{base58.Encode(p.pk)},
			ServiceEndpoint: ae.Endp,
		}}),
	))
	return doc
}

// newPeerDID is copied from framework's tests to find smallest common divider
// to create `did:peer` with only one dependcy which is here kms.KeyManager.
func _(keys kms.KeyManager) (d *did.Doc, err error) {
	defer err2.Return(&err)

	kid, pubKey, err := keys.CreateAndExportPubKeyBytes(kms.ED25519)
	err2.Check(err)

	key := did.VerificationMethod{
		ID:         kid,
		Type:       "Ed25519VerificationKey2018",
		Controller: "",
		Value:      pubKey,
	}
	doc, err := peer.NewDoc(
		[]did.VerificationMethod{key},
		did.WithAuthentication([]did.Verification{{
			VerificationMethod: key,
			Relationship:       0,
			Embedded:           true,
		}}),
		did.WithService([]did.Service{{
			ID:              "didcomm",
			Type:            "did-communication",
			Priority:        0,
			RecipientKeys:   []string{base58.Encode(pubKey)},
			ServiceEndpoint: "http://example.com",
		}}),
	)
	err2.Check(err)

	return doc, nil
}
