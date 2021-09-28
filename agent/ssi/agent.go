package ssi

import (
	"fmt"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/findy-network/findy-wrapper-go/ledger"
	"github.com/findy-network/findy-wrapper-go/pairwise"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type AgentType interface {
	IsCA() bool
	IsEA() bool
}

type Agent interface {
	AgentType
	Wallet() (h int)
	RootDid() *DID
	CreateDID(seed string) (agentDid *DID)
	SendNYM(targetDid *DID, submitterDid, alias, role string) error
	AddDIDCache(DID *DID)
}

// Type of the agent instance. In most cases it's Cloud Agent (CA). Which is the
// the default value.
type Type int

// Please be noted that Cloud Agent is the default value.
const (
	// Edge agents are the agents which are at the end of the agent route. They
	// are the final endpoint of the agent messages. In the agency we can have
	// pure EAs when CLI is used, or we can have Worker EAs which are working
	// together with their Cloud Agent.
	Edge = 0x01

	// Worker is an Edge Agent in the Cloud. Workers are used to allow EAs to
	// have endpoints inside to identity domain. Worker EAs can be always on,
	// and listen their endpoints. These cloud EAs have their own wallets,
	// which can be copied to actual EA's device if needed.
	Worker = 0x02
)

/*
DIDAgent is the main abstraction of the package together with Agency. The agent
started as a CA but has been later added support for EAs and worker/cloud-EA as
well. This might be something we will change later. DIDAgent's most important
task is/WAS to receive Payloads and process Messages inside them. And there are
lots of stuff to support that. That part of code is heavily under construction.

More concrete parts of the DIDAgent are support for wallet, root DID, did cache.
Web socket connections are more like old relic, and that will change in future
for something else. It WAS part of the protocol STATE management.

Please be noted that DIDAgent or more precisely CA is singleton by its nature
per EA it serves. So, Cloud DIDAgent is a gateway to world for EA it serves. EAs
are mostly in mobile devices and handicapped by their nature. In our latest
architecture CA serves EA by creating a worker EA which lives in the cloud as
well. For now, in the most cases we have pair or agents serving each mobile EAs
here in the cloud: CA and w-EA.

There is DIDAgent.Type where this DIDAgent can be EA only. That type is used for
test and CLI Go clients.
*/
type DIDAgent struct {
	WalletH managed.Wallet

	// result future of the wallet export, one time attr, obsolete soon
	Export Future

	// the Root DID which gives us rights to write ledger
	Root *DID
	// keep 'all' DIDs for performance reasons as well as better usability of our APIs
	DidCache Cache
	// Agent type: CA, EA, Worker, etc.
	Type Type

	SAImplID string        // SA implementation ID, used mostly for tests
	EAEndp   *service.Addr // EA endpoint if set, used for SA API and notifications
}

func (a *DIDAgent) AddDIDCache(DID *DID) {
	a.DidCache.Add(DID)
}

func (a *DIDAgent) IsCA() bool {
	// Our default agent type is Cloud DIDAgent but we don't want to set zero to
	// Cloud type. Instead we state that if we are not EA we are CA.
	return !a.IsEA()
}

func (a *DIDAgent) IsEA() bool {
	return a.Type&Edge != 0
}

func (a *DIDAgent) IsWorker() bool {
	return a.Type&Worker != 0
}

func (a *DIDAgent) AssertWallet() {
	if a.WalletH == nil {
		panic("Fatal Programming Error!")
	}
}

func (a *DIDAgent) assertPool() {
	if a.Pool() == 0 {
		panic("Fatal Programming Error!")
	}
}

// MARK: Wallet --

func (a *DIDAgent) OpenWallet(aw Wallet) {
	a.WalletH = Wallets.Open(&aw)
	if glog.V(5) {
		glog.Info("Opening wallet: ", aw.Config.ID)
	}
}

func (a *DIDAgent) CloseWallet() {
	if a.WalletH != nil {
		a.WalletH.Close()
	} else {
		glog.Warning("no wallet to close!")
	}
}

func (a *DIDAgent) Wallet() (h int) {
	return a.WalletH.Handle()
}

// MARK: Pool --

func (a *DIDAgent) OpenPool(name string) {
	OpenPool(name)
}

func (a *DIDAgent) Pool() (v int) {
	return Pool()
}

// MARK: DID --

// CreateDID creates a new DID thru the Future which means that returned *DID
// follows 'lazy fetch' principle. You should call this as early as possible for
// the performance reasons. Most cases seed should be empty string.
func (a *DIDAgent) CreateDID(seed string) (agentDid *DID) {
	a.AssertWallet()
	f := new(Future)
	f.SetChan(did.CreateAndStore(a.Wallet(), did.Did{Seed: seed}))
	return NewAgentDid(a.WalletH, f)
}

func (a *DIDAgent) RootDid() *DID {
	return a.Root
}

func (a *DIDAgent) SetRootDid(rootDid *DID) {
	a.Root = rootDid
}

// MARK: App logic

func (a *DIDAgent) SendNYM(
	targetDid *DID, submitterDid, alias, role string) (err error) {

	a.AssertWallet()
	a.assertPool()
	targetDID := targetDid.Did()
	verkey := targetDid.VerKey()
	return ledger.WriteDID(a.Pool(), a.Wallet(), submitterDid, targetDID, verkey, alias, role)
}

// localKey returns a future to the verkey of the DID from a local wallet.
func (a *DIDAgent) localKey(didName string) (f *Future) {
	f = new(Future)
	f.SetChan(did.LocalKey(a.Wallet(), didName))
	return
}

func (a *DIDAgent) SaveTheirDID(did, vk string, writeNYM bool) (err error) {
	defer err2.Return(&err)

	newDID := NewDid(did, vk)
	a.DidCache.Add(newDID)
	newDID.Store(a.Wallet())

	// Previous is an async func so make sure results are ready
	// check the errors now before writing to the ledger
	err2.Check(newDID.StoreResult())

	if writeNYM && !utils.Settings.LocalTestMode() {
		err2.Check(a.SendNYM(newDID, a.RootDid().Did(),
			findy.NullString, findy.NullString))
	}
	return nil
}

func (a *DIDAgent) OpenDID(name string) *DID {
	verkey := a.localKey(name)
	newDid := NewDid(name, verkey.Str1())
	a.DidCache.LazyAdd(name, newDid)
	return newDid
}

func (a *DIDAgent) LoadDID(did string) *DID {
	cached := a.DidCache.Get(did, true)
	if cached != nil {
		if cached.Wallet() == 0 {
			cached.SetWallet(a.WalletH)
		}
		//log.Println("Return cached DID")
		return cached
	}
	d := NewDidWithKeyFuture(a.WalletH, did, a.localKey(did))
	a.DidCache.Add(d)
	return d
}

func (a *DIDAgent) Pairwise(name string) (my string, their string, err error) {
	a.AssertWallet()
	r := <-pairwise.List(a.Wallet())
	if r.Err() != nil {
		return "", "", fmt.Errorf("agent pairwise: %s", r.Err())
	}
	pwd := pairwise.NewData(r.Str1())

	for _, d := range pwd {
		if d.Metadata == name || name == "" {
			return d.MyDid, d.TheirDid, nil
		}
	}
	return "", "", nil
}

// FindPW finds pairwise by name. This is a ReceiverEndp interface method.
func (a *DIDAgent) FindPW(my string) (their string, pwname string, err error) {
	a.AssertWallet()
	r := <-pairwise.List(a.Wallet())
	if r.Err() != nil {
		return "", "", fmt.Errorf("agent pairwise: %s", r.Err())
	}
	pwd := pairwise.NewData(r.Str1())
	for _, d := range pwd {
		if d.MyDid == my {
			return d.TheirDid, d.Metadata, nil
		}
	}
	return "", "", nil
}
