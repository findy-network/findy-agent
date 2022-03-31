package ssi

import (
	"fmt"
	"sync"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/storage/api"
	storage "github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/indy"
	dto "github.com/findy-network/findy-common-go/dto"
	"github.com/findy-network/findy-wrapper-go/did"
	indyDto "github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type DidComm interface {
	Did() string
	Storage() managed.Wallet
}

type Out interface {
	DidComm
	VerKey() string
	Route() []string
	Endpoint() string                    // refactor
	AEndp() (ae service.Addr, err error) // refactor
}

type In interface {
	Out
	Wallet() int
}

// DID is an application framework level wrapper for findy.DID implementation.
// Uses Future to async processing of the findy.Channel results.
type DID struct {
	// Wallet handle if available.
	// Implementation Note: we will build access with these handles to the Indy
	// AgentStorage. managed.WalletCfg.UniqueID() will be the key.
	wallet managed.Wallet

	data   *Future // DID data when queried from the wallet or received somewhere
	stored *Future // result of the StartStore Their DID operation
	key    *Future // DID construct where key is Future
	meta   *Future // Meta data stored to DID
	pw     *Future // Pairwise data stored to DID
	endp   *Future // When endpoint is started to fetch it's here

	sync.Mutex // when setting Future ptrs making sure that happens atomically

	pwMeta *PairwiseMeta // Meta data for pairwise

}

func (d *DID) Storage() managed.Wallet {
	return d.wallet
}

func (d *DID) Packager() api.Packager {
	return AgentStorage(d.wallet.Handle()).OurPackager()
}

func (d *DID) KMS() *indy.KMS {
	return AgentStorage(d.wallet.Handle()).OurPackager().KMS().(*indy.KMS)
}

// String returns a string in DID format e.g. 'did:sov:xxx..'
func (d *DID) String() string {
	return indy.MethodPrefix + d.Did()
}

// KID returns a KMS specific key ID that can be used to Get KH from KMS.
func (d *DID) KID() string {
	return d.Did()
}

// SignKey return a indy.Handle including wallet SDK handle (int) and a VerKey
// TODO: Let's think if wee need a KID for there as well
func (d *DID) SignKey() any {
	return &indy.Handle{
		Wallet: d.Wallet(),
		VerKey: d.VerKey(),
	}
}

type PairwiseMeta struct {
	Name  string
	Route []string
}

func NewAgentDid(wallet managed.Wallet, f *Future) (ad *DID) {
	d := &DID{wallet: wallet, data: f}
	d.SetWallet(wallet)
	return d
}

func NewDid(did, verkey string) (d *DID) {
	f := &Future{V: indyDto.Result{Data: indyDto.Data{Str1: did, Str2: verkey}}, On: Consumed}
	return &DID{data: f}
}

func NewOutDid(verkey string, route []string) (d *DID) {
	did := NewDid("", verkey)
	did.pwMeta = &PairwiseMeta{Route: route}
	return did
}

func NewDidWithKeyFuture(wallet managed.Wallet, did string, verkey *Future) (d *DID) {
	f := &Future{V: indyDto.Result{Data: indyDto.Data{Str1: did, Str2: ""}}, On: Consumed}
	d = &DID{wallet: wallet, data: f, key: verkey}
	d.SetWallet(wallet)
	return d
}

func (d *DID) Did() string {
	didStr, _, _ := d.data.Strs()
	return didStr
}

func (d *DID) URI() string {
	return "did:sov:" + d.Did()
}

func (d *DID) VerKey() (vk string) {
	if d.hasKeyData() {
		_, vk, _ = d.data.Strs()
	} else if d.key != nil {
		vk = d.key.Str1()
	} else {
		vk = ""
	}
	return vk
}

func (d *DID) Wallet() int {
	if d.wallet == nil {
		return 0
	}
	return d.wallet.Handle()
}

func (d *DID) SetWallet(w managed.Wallet) {
	d.wallet = w
	if d.Did() != "" && d.VerKey() != "" {
		d.KMS().Add(d.Did(), d.VerKey())
	}
}

// Store stores this DID as their DID to given wallet. Work is done thru futures
// so the call doesn't block. The meta data is set "pairwise". See StoreResult()
// for status.
func (d *DID) Store(mgdWallet, mgdStorage managed.Wallet) {
	defer err2.Catch(func(err error) {
		glog.Errorf("Error storing DID: %s", err)
	})

	ds, vk, _ := d.data.Strs()
	idJSON := did.Did{Did: ds, VerKey: vk}
	json := dto.ToJSON(idJSON)

	f := new(Future)
	f.SetChan(did.StoreTheir(mgdWallet.Handle(), json))

	// Store the DID also to the agent storage
	glog.V(5).Infof("agent storage Store DID %s", ds)
	try.To(mgdStorage.Storage().DIDStorage().SaveDID(storage.DID{
		ID:         ds,
		DID:        ds,
		IndyVerKey: vk,
	}))

	d.Lock()
	// we use stored lock just for extra safety. The whole indy.DID implementation
	// will change
	if d.wallet == nil {
		d.SetWallet(mgdWallet)
	}
	d.stored = f
	d.Unlock()
}

// StoreResult returns error status of the Store() functions result. If storing
// their DID and related meta and pairwise data isn't ready, this call blocks.
func (d *DID) StoreResult() error {
	d.Lock()
	stored := d.stored
	d.Unlock()

	if stored != nil && stored.Result().Err() != nil {
		return fmt.Errorf("their: %s", stored.Result().Error())
	}

	d.Lock()
	meta := d.meta
	d.Unlock()

	if meta != nil && meta.Result().Err() != nil {
		return fmt.Errorf("meta: %s", meta.Result().Error())
	}

	d.Lock()
	pw := d.pw
	d.Unlock()

	if pw != nil && pw.Result().Err() != nil {
		return fmt.Errorf("pairwise: %s", pw.Result().Error())
	}

	return nil
}

func (d *DID) SavePairwiseForDID(mStorage managed.Wallet, theirDID *DID, pw PairwiseMeta) {
	defer err2.Catch(func(err error) {
		glog.Warningf("save pairwise for DID error: %v", err)
	})

	// check that DIDs are ready
	ok := d.data.Result().Err() == nil && theirDID.stored.Result().Err() == nil
	if ok {
		connection, _ := mStorage.Storage().ConnectionStorage().GetConnection(pw.Name)
		if connection == nil {
			connection = &storage.Connection{
				ID: pw.Name,
			}
		}
		connection.MyDID = d.Did()
		connection.TheirDID = theirDID.Did()
		connection.TheirRoute = pw.Route
		glog.V(7).Infoln("=== save connection:",
			connection.ID, connection.MyDID, connection.TheirDID)

		err := mStorage.Storage().ConnectionStorage().SaveConnection(*connection)
		errStr := ""
		if err != nil {
			ok = false
			errStr = err.Error()
		}

		f := &Future{V: indyDto.Result{Er: indyDto.Err{Error: errStr}}, On: Consumed}
		theirDID.pw = f
	}
	if !ok {
		glog.Error("Could not store pairwise info")
	}
}

func (d *DID) hasKeyData() bool {
	_, vk, _ := d.data.Strs()
	return vk != ""
}

func (d *DID) StartEndp(storageH managed.Wallet, connectionID string) {
	store := storageH.Storage().ConnectionStorage()
	connection, err := store.GetConnection(connectionID)
	endpoint := ""
	errStr := ""
	if err == nil {
		endpoint = connection.TheirEndpoint
	} else {
		glog.Warningf("--- get connection (%s) failure", connectionID)
		errStr = err.Error()
	}

	f := &Future{V: indyDto.Result{
		Data: indyDto.Data{Str1: endpoint},
		Er:   indyDto.Err{Error: errStr},
	}, On: Consumed}

	d.Lock()
	d.endp = f
	d.Unlock()
}

func (d *DID) Endpoint() string {
	defer func() {
		if r := recover(); r != nil {
			glog.Warning("Recovered in did.endpoint", r)
		}
	}()

	d.Lock()
	endp := d.endp
	d.Unlock()

	if endp != nil && endp.Result().Err() != nil {
		return ""
	} else if endp != nil {
		return endp.Str1()
	}
	return ""
}

func (d *DID) SetAEndp(ae service.Addr) {
	d.endp = &Future{
		V:  indyDto.Result{Data: indyDto.Data{Str1: ae.Endp, Str2: ae.Key}},
		On: Consumed,
	}
}

func (d *DID) AEndp() (ae service.Addr, err error) {
	d.Lock()
	endp := d.endp
	d.Unlock()

	if endp != nil && endp.Result().Err() != nil {
		return service.Addr{}, endp.Result().Err()
	} else if endp != nil {
		endP, vk, _ := endp.Strs()
		return service.Addr{Endp: endP, Key: vk}, nil
	}
	return service.Addr{}, fmt.Errorf("no data")
}

func (d *DID) Route() []string {
	if d.pwMeta != nil {
		return d.pwMeta.Route
	}
	return []string{}
}
