package ssi

import (
	"sync"

	"github.com/findy-network/findy-agent/method"
	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
	storage "github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/findy-network/findy-agent/agent/vdr"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/did"
	indyDto "github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/ledger"
	"github.com/golang/glog"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type AgentType interface {
	IsCA() bool
	IsEA() bool
}

type Agent interface {
	AgentType
	Wallet() (h int)
	ManagedWallet() managed.Wallet
	RootDid() *DID
	CreateDID(seed string) (agentDid *DID)
	NewDID(method string) core.DID
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

	sync.Mutex // Currently saImplID makes the agent mutable

	saImplID string        // SA implementation ID, used mostly for tests
	EAEndp   *service.Addr // EA endpoint if set, used for SA API and notifications
}

func (a *DIDAgent) SAImplID() string {
	a.Lock()
	defer a.Unlock()
	return a.saImplID
}

func (a *DIDAgent) SetSAImplID(id string) {
	a.Lock()
	defer a.Unlock()
	a.saImplID = id
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

func (a *DIDAgent) OpenWallet(aw Wallet) {
	a.WalletH = wallets.Open(aw)
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

func (a *DIDAgent) ManagedWallet() managed.Wallet {
	return a.WalletH
}

func (a *DIDAgent) OpenPool(name string) {
	OpenPool(name)
}

func (a *DIDAgent) Pool() (v int) {
	return Pool()
}

func (a *DIDAgent) VDR() *vdr.VDR {
	apiStorage := a.ManagedWallet().Storage()

	as, ok := apiStorage.(*mgddb.Storage)
	assert.D.True(ok, "todo: update type later!!")

	return try.To1(vdr.New(as))
}

func (a *DIDAgent) NewDID(didmeth string) core.DID {
	switch didmeth {
	case "key":
		// we will used the correct VDR to create the correct did
		// the VDR is the factory for a DID method
		_ = a.VDR()
		kms := a.KMS()
		return try.To1(method.NewKey(kms))

	case "indy":
		return a.CreateDID("")
	default:
		return a.CreateDID("")
		//assert.That(false, "not supported")

	}
	return nil
}

// CreateDID creates a new DID thru the Future which means that returned *DID
// follows 'lazy fetch' principle. You should call this as early as possible for
// the performance reasons. Most cases seed should be empty string.
func (a *DIDAgent) CreateDID(seed string) (agentDid *DID) {
	a.AssertWallet()
	f := new(Future)
	ch := make(findy.Channel, 1)
	go func() {
		defer err2.Catch(func(err error) {
			glog.Errorf("AddDID failed: %s", err)
		})
		// Catch did result here and store it also to the agent storage
		didRes := <-did.CreateAndStore(a.Wallet(), did.Did{Seed: seed})
		glog.V(5).Infof("agent storage Add DID %s", didRes.Data.Str1)
		try.To(a.DIDStorage().SaveDID(storage.DID{
			ID:         didRes.Data.Str1,
			DID:        didRes.Data.Str1,
			IndyVerKey: didRes.Data.Str2,
		}))
		ch <- didRes
	}()
	f.SetChan(ch)
	return NewAgentDid(a.WalletH, f)
}

func (a *DIDAgent) RootDid() *DID {
	return a.Root
}

func (a *DIDAgent) SetRootDid(rootDid *DID) {
	a.Root = rootDid
}

func (a *DIDAgent) SendNYM(
	targetDid *DID,
	submitterDid,
	alias,
	role string,
) (err error) {
	a.AssertWallet()
	a.assertPool()

	targetDID := targetDid.Did()
	verkey := targetDid.VerKey()
	return ledger.WriteDID(a.Pool(), a.Wallet(), submitterDid, targetDID, verkey, alias, role)
}

func (a *DIDAgent) ConnectionStorage() storage.ConnectionStorage {
	return a.ManagedWallet().Storage().ConnectionStorage()
}

func (a *DIDAgent) DIDStorage() storage.DIDStorage {
	return a.ManagedWallet().Storage().DIDStorage()
}

func (a *DIDAgent) KMS() kms.KeyManager {
	return a.ManagedWallet().Storage().KMS()
}

// localKey returns a future to the verkey of the DID from a local wallet.
func (a *DIDAgent) localKey(didName string) (f *Future) {
	defer err2.Catch(func(err error) {
		glog.Errorln("error when fetching localKey: ", err)
	})

	// using did storage to get the verkey - could be also fetched from indy wallet directly
	// eventually all data should be fetched from agent storage and not from indy wallet
	did := try.To1(a.DIDStorage().GetDID(didName))

	glog.V(5).Infoln("found localKey: ", didName, did.IndyVerKey)

	return &Future{V: indyDto.Result{Data: indyDto.Data{Str1: did.IndyVerKey}}, On: Consumed}
}

func (a *DIDAgent) SaveTheirDID(did, vk string) (err error) {
	defer err2.Return(&err)

	newDID := NewDid(did, vk)
	a.DidCache.Add(newDID)
	newDID.Store(a.ManagedWallet())

	// Previous is an async func so make sure results are ready
	try.To(newDID.StoreResult())

	return nil
}

// Used by steward only
func (a *DIDAgent) OpenDID(name string) *DID {
	f := new(Future)
	f.SetChan(did.LocalKey(a.Wallet(), name))

	newDid := NewDid(name, f.Str1())
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

func (a *DIDAgent) LoadTheirDID(connection storage.Connection) *DID {
	did := a.LoadDID(connection.TheirDID)
	did.pwMeta = &PairwiseMeta{Route: connection.TheirRoute}
	return did
}

func (a *DIDAgent) FindPWByName(name string) (pw *storage.Connection, err error) {
	a.AssertWallet()
	return a.ConnectionStorage().GetConnection(name)
}

// FindPWByDID finds pairwise by my DID. This is a ReceiverEndp interface method.
func (a *DIDAgent) FindPWByDID(my string) (pw *storage.Connection, err error) {
	defer err2.Catch(func(err error) {
		glog.Error("cannot find pw by id:", err)
	})

	a.AssertWallet()

	connections := try.To1(a.ConnectionStorage().ListConnections())

	for _, item := range connections {
		if item.MyDID == my {
			return &item, nil
		}
	}
	return nil, nil
}
