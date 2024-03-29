package method

import (
	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/core"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/lainio/err2/assert"
	"github.com/mr-tron/base58"
)

type Base struct {
	kid string
	pk  []byte
	vkh any

	//doc *did.Doc
	doc core.DIDDoc

	handle managed.Wallet
}

// String returns URI formatted DID
func (b Base) String() string {
	// TODO: lazy fetch or move to constructor, ANSWER: cheap to calc
	didkey, _ := fingerprint.CreateDIDKey(b.pk)

	return didkey
}

func (b Base) URI() string {
	return b.String()
}

// Did function returns current KID of it exists.
func (b Base) Did() string {
	return b.URI()
}

func (b Base) KID() string {
	return b.kid
}

func (b Base) VerKey() string {
	return base58.Encode(b.pk)
}

func (b Base) Packager() api.Packager {
	return b.handle.Storage().OurPackager()
}

func (b Base) DOC() core.DIDDoc {
	return b.doc
}

func (b Base) SignKey() any {
	return b.vkh
}

func (b Base) Storage() managed.Wallet {
	return b.handle
}

func (b Base) StartEndp(_ managed.Wallet, _ string) {
	// todo: check how this is implemented in ssi.DID.
	// It seems that it's simple, but unnesseary.
}

func (b Base) Store(_, _ managed.Wallet) {
	// todo: check the implementation from ssi.DID
	// it seems that there is nothing to do, all is saved already.
}

func (b Base) SavePairwiseForDID(_ managed.Wallet, _ core.DID, _ core.PairwiseMeta) {
	// nothing to do at this level because we don't have virtual functions
	// the implementation must be done at the lower impl level like Peer
}

func (b Base) StoreResult() error {
	// todo: see ssi.DID
	return nil
}

func (b Base) AEndp() (ae service.Addr, err error) {
	assert.NotImplemented()
	return
}

func (b Base) SetAEndp(_ service.Addr) {
	// Key don't use and peer has its document already
}
