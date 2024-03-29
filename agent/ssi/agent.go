package ssi

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/findy-network/findy-agent/agent/async"
	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/pool"
	"github.com/findy-network/findy-agent/agent/service"
	storage "github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/cfg"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/agent/vdr"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/indy"
	"github.com/findy-network/findy-agent/method"
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
	ManagedWallet() (managed.Wallet, managed.Wallet)
	RootDid() core.DID
	// CreateDID(seed string) (agentDid core.DID)
	NewDID(m method.Type, args ...string) (_ core.DID, err error)
	NewOutDID(didInfo ...string) (id core.DID, err error)
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
	WalletH  managed.Wallet
	StorageH managed.Wallet

	// result future of the wallet export, one time attr, obsolete soon
	Export async.Future

	// the Root DID which gives us rights to write ledger
	Root core.DID

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
	assert.That(a.WalletH != nil && a.StorageH != nil)
}

func (a *DIDAgent) assertPool() {
	if a.Pool() == 0 {
		panic("Fatal Programming Error!")
	}
}

func (a *DIDAgent) OpenWallet(aw Wallet) {
	c := new(Wallet)
	*c = aw
	a.WalletH = wallets.Open(c)
	if glog.V(5) {
		glog.Info("Opening wallet: ", aw.Config.ID)
	}

	path := utils.IndyBaseDir()
	path = filepath.Join(path, "storage")
	sc := &cfg.AgentStorage{AgentStorageConfig: storage.AgentStorageConfig{
		AgentKey: generateKey(),
		AgentID:  aw.ID(),
		FilePath: path,
	}}
	a.StorageH = storages.Open(sc)
}

func generateKey() string {
	// TODO
	return "15308490f1e4026284594dd08d31291bc8ef2aeac730d0daf6ff87bb92d4336c"
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

func (a *DIDAgent) ManagedWallet() (managed.Wallet, managed.Wallet) {
	return a.WalletH, a.StorageH
}

func (a *DIDAgent) ManagedStorage() managed.Wallet {
	return a.StorageH
}

// Storage returns TEMPORARY agent storage object pointer. Note!! You should
// newer store it, only use by once, even in every single line of code.
func (a *DIDAgent) Storage() storage.AgentStorage {
	return a.ManagedStorage().Storage()
}

func (a *DIDAgent) OpenPool(name string) {
	pool.Open(name)
}

func (a *DIDAgent) Pool() (v int) {
	return pool.Handle()
}

func (a *DIDAgent) VDR() *vdr.VDR {
	aStorage, ok := a.Storage().(*mgddb.Storage)
	assert.That(ok, "TODO: update type later!!")

	return try.To1(vdr.New(aStorage))
}

func (a *DIDAgent) NewDID(didMethod method.Type, args ...string) (_ core.DID, err error) {
	defer err2.Handle(&err, "new DID")

	glog.V(3).Infof("New '%s': %v", didMethod.DIDString(), args)

	switch didMethod {
	case method.TypeKey, method.TypePeer:
		// _ = a.VDR() // TODO: check if we could use VDR as method factory
		coreDID := try.To1(method.New(didMethod, a.StorageH, args...))
		try.To(a.saveDID(coreDID))
		return coreDID, err

	case method.TypeSov, method.TypeIndy:
		return a.myCreateDID(args[0]), err
	default:
		return a.myCreateDID(args[0]), err // TODO: remove after test
		// assert.That(false, "not supported")
	}
}

func (a *DIDAgent) saveDID(coreDID core.DID) (err error) {
	defer err2.Handle(&err)

	sDID := storage.DID{
		KID: coreDID.KID(),
		ID:  coreDID.URI(),
		DID: coreDID.URI(),
		Doc: try.To1(json.Marshal(coreDID.DOC())),
	}
	try.To(a.DIDStorage().SaveDID(sDID))
	glog.V(3).Infoln("did saved:", sDID.KID, sDID.ID)
	return nil
}

func (a *DIDAgent) NewOutDID(didInfo ...string) (id core.DID, err error) {
	defer err2.Handle(&err, "new out DID")

	switch method.DIDType(didInfo[0]) {
	case method.TypeKey, method.TypePeer:
		coreDID := try.To1(method.NewFromDID(
			a.StorageH,
			didInfo...,
		))
		try.To(a.saveDID(coreDID))
		return coreDID, nil

	case method.TypeIndy, method.TypeSov:
		assert.That(len(didInfo) >= 2)

		glog.V(10).Infof("NewOutDID(didstr= %s, verKey= %s)",
			didInfo[0], didInfo[1])

		d := indy.DID2KID(didInfo[0])
		var cached *DID
		if d != "" {
			try.To(a.SaveTheirDID(d, didInfo[1]))
			cached = a.DidCache.Get(d, true)
			assert.That(cached.wallet != nil)
		} else {
			newDID := NewDIDWithRouting("", didInfo[1:]...)
			a.DidCache.Add(newDID)
			cached = newDID
		}
		return cached, nil
	default:
		assert.That(false, "did method not supported: %s", didInfo[0])
		return nil, nil
	}
}

// myCreateDID creates a new DID thru the Future which means that returned *DID
// follows 'lazy fetch' principle. You should call this as early as possible for
// the performance reasons. Most cases seed should be empty string.
func (a *DIDAgent) myCreateDID(seed string) (agentDid *DID) {
	a.AssertWallet()

	// If endpoint is sent to us instead of seed, we must ignore it.
	// The actual endpoint will be sent to us when it's ready.
	ap := seed
	if u, err := url.Parse(seed); err == nil && u.Scheme != "" {
		glog.V(10).Infoln("Seed is URL:", ap)
		seed = ""
	} else {
		glog.V(10).Infoln("Seed is EXT-seed, using external Steward!")
		ap = ""
	}

	f := new(async.Future)
	ch := make(findy.Channel, 1)
	go func() {
		defer err2.Catch(err2.Err(func(err error) {
			glog.Errorf("AddDID failed: %s", err)
		}))
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
	d := NewAgentDid(a.WalletH, f)
	if ap != "" {
		glog.V(10).Infoln("Setting Endpoint to sov.did")
		d.SetAEndp(service.Addr{
			Endp: ap,
			Key:  d.VerKey(),
		})
	}
	return d
}

func (a *DIDAgent) RootDid() core.DID {
	return a.Root
}

func (a *DIDAgent) SetRootDid(rootDid core.DID) {
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
	return a.Storage().ConnectionStorage()
}

func (a *DIDAgent) DIDStorage() storage.DIDStorage {
	return a.Storage().DIDStorage()
}

func (a *DIDAgent) KMS() kms.KeyManager {
	return a.Storage().KMS()
}

// localKey returns a future to the verkey of the DID from a local wallet.
func (a *DIDAgent) localKey(didName string) (f *async.Future) {
	defer err2.Catch(err2.Err(func(err error) {
		glog.Errorln("error when fetching localKey: ", err)
	}))

	// using did storage to get the verkey - could be also fetched from indy wallet directly
	// eventually all data should be fetched from agent storage and not from indy wallet
	d := try.To1(a.DIDStorage().GetDID(didName))

	glog.V(5).Infoln("found localKey: ", didName, d.IndyVerKey)

	return &async.Future{V: indyDto.Result{Data: indyDto.Data{Str1: d.IndyVerKey}}, On: async.Consumed}
}

func (a *DIDAgent) SaveTheirDID(did, vk string) (err error) {
	defer err2.Handle(&err)

	newDID := NewDid(did, vk)
	a.DidCache.Add(newDID)
	newDID.Store(a.ManagedWallet())

	// Previous is an async func so make sure results are ready
	try.To(newDID.StoreResult())

	return nil
}

// OpenDID NOTE! Used by steward only.
func (a *DIDAgent) OpenDID(name string) *DID {
	f := new(async.Future)
	f.SetChan(did.LocalKey(a.Wallet(), name))

	newDid := NewDid(name, f.Str1())
	a.DidCache.LazyAdd(name, newDid)
	return newDid
}

func (a *DIDAgent) LoadDID(did string) core.DID {
	switch method.DIDType(did) {
	case method.TypePeer:
		storageDID := try.To1(a.DIDStorage().GetDID(did))
		return try.To1(method.NewPeerFromDID(a.StorageH, storageDID))

	default:
		cached := a.DidCache.Get(did, true)
		if cached != nil {
			if cached.Wallet() == 0 {
				cached.SetWallet(a.WalletH)
			}
			// log.Println("Return cached DID")
			return cached
		}
		d := NewDidWithKeyFuture(a.WalletH, did, a.localKey(did))
		a.DidCache.Add(d)
		return d
	}
}

func (a *DIDAgent) LoadTheirDID(connection storage.Connection) core.DID {
	defer err2.Catch(err2.Err(func(err error) {
		glog.Warningf("load connection (%s) error: %v", connection.ID, err)
	}), func(v any) {
		glog.Warningf("load connection (%s) error: %v", connection.ID, v)
	})

	assert.That(connection.TheirDID != "")

	d := a.LoadDID(connection.TheirDID)
	// TODO: implement!
	// d.pwMeta = &PairwiseMeta{Route: connection.TheirRoute}
	return d
}

func (a *DIDAgent) FindPWByName(name string) (pw *storage.Connection, err error) {
	a.AssertWallet()
	assert.That(name != "")
	return a.ConnectionStorage().GetConnection(name)
}

// FindPWByID finds pairwise by id. This is a ReceiverEndp interface method.
func (a *DIDAgent) FindPWByID(id string) (pw *storage.Connection, err error) {
	return a.FindPWByName(id)
}
