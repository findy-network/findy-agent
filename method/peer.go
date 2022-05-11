package method

import (
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

type Peer struct {
	Base
}

func NewPeerFromDID(
	hStorage managed.Wallet,
	d *api.DID,
) (
	id core.DID,
	err error,
) {
	defer err2.Return(&err)

	doc := d.Doc.DIDDocument
	pk := doc.VerificationMethod[0].Value
	keys := hStorage.Storage().KMS()
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Peer{Base{handle: hStorage, kid: d.KID, pk: pk, vkh: kh, doc: doc}},
		nil
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
		ID:         "1",
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

// NewPeerFromDoc doesn't create a totally new did:peer but it saves its pubkey
// to our kms for us to be able to use cryptos with them.
func NewPeerFromDoc(
	hStorage managed.Wallet,
	didDoc string,
) (
	id core.DID,
	err error,
) {
	defer err2.Annotate("new did:peer from did", &err)

	doc := try.To1(did.ParseDocument([]byte(didDoc)))
	pk := doc.VerificationMethod[0].Value
	keys := hStorage.Storage().KMS()
	kh := try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519))

	return Peer{Base{handle: hStorage, kid: "", pk: pk, vkh: kh, doc: doc}}, nil
}

func (p Peer) NewDoc(ae service.Addr) core.DIDDoc {
	if p.doc != nil {
		return p.doc
	}

	myAE, _ := p.AEndp()
	assert.That(ae == myAE)

	key := did.VerificationMethod{
		ID:         "1",
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

// String returns URI formated 'did:key:' for verification method
func (p Peer) String() string {
	return p.buildDIDKeyStr(p.VerKey())
}

func (p Peer) VerKey() string {
	assert.NotNil(p.doc)
	assert.SNotEmpty(p.doc.VerificationMethod)

	vk := base58.Encode(p.doc.VerificationMethod[0].Value)
	return vk
}

func (p Peer) Route() []string {
	assert.NotNil(p.doc)
	doc := p.doc

	services := doc.Service[0]
	route := make([]string, len(services.RoutingKeys))
	for i, rk := range services.RoutingKeys {
		route[i] = p.buildDIDKeyStr(rk)
	}
	return route
}

func (p Peer) RecipientKeys() []string {
	assert.NotNil(p.doc)
	doc := p.doc

	services := doc.Service[0]
	route := make([]string, len(services.RecipientKeys))
	for i, rk := range services.RecipientKeys {
		route[i] = p.buildDIDKeyStr(rk)
	}
	return route
}

func (p Peer) Did() string {
	assert.NotNil(p.doc)
	return p.doc.ID
}

func (p Peer) URI() string {
	assert.NotNil(p.doc)
	return p.doc.ID
}

func (b Base) buildDIDKeyStr(rk string) string {
	keys := b.handle.Storage().KMS()
	pk := try.To1(base58.Decode(rk))
	_ = try.To1(keys.PubKeyBytesToHandle(pk, kms.ED25519)) // we don't need handle
	didkey, _ := fingerprint.CreateDIDKey(pk)
	return didkey
}

func (p Peer) AEndp() (ae service.Addr, err error) {
	assert.NotNil(p.doc)
	return service.Addr{
		Endp: p.doc.Service[0].ServiceEndpoint,
		Key:  p.doc.Service[0].RecipientKeys[0],
	}, nil
}

func (p Peer) SavePairwiseForDID(mStorage managed.Wallet, theirDID core.DID,
	pw core.PairwiseMeta) {
	defer err2.Catch(func(err error) {
		glog.Warningf("save pairwise for DID error: %v", err)
	})

	connection, _ := mStorage.Storage().ConnectionStorage().GetConnection(pw.Name)

	if connection == nil {
		connection = &api.Connection{
			ID: pw.Name,
		}
	}
	connection.MyDID = p.Did()
	connection.TheirDID = theirDID.Did()
	connection.TheirRoute = pw.Route
	glog.V(7).Infoln("=== save connection:",
		connection.ID, connection.MyDID, connection.TheirDID)

	try.To(mStorage.Storage().ConnectionStorage().SaveConnection(*connection))
}

func NewDoc(pk, addr string) (d *did.Doc, err error) {
	defer err2.Return(&err)

	pubKey := try.To1(base58.Decode(pk))

	key := did.VerificationMethod{
		ID:         "1", // TODO: now just base58 pubkey
		Type:       "Ed25519VerificationKey2018",
		Controller: "",
		Value:      pubKey,
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
			RecipientKeys:   []string{base58.Encode(pubKey)},
			ServiceEndpoint: addr,
		}}),
	))

	return doc, nil
}
