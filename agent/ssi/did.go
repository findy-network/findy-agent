package ssi

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/pairwise"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type DidComm interface {
	Did() string
}

type Out interface {
	DidComm
	VerKey() string
	Route() []string
	Endpoint() string                      // refactor
	AEndp() (ae service.Addr, error error) // refactor
}

type In interface {
	Out
	Wallet() int
}

// DID is an application framework level wrapper for findy.DID implementation.
// Uses Future to async processing of the findy.Channel results.
type DID struct {
	wallet managed.Wallet // Wallet handle if available

	data   *Future // DID data when queried from the wallet or received somewhere
	stored *Future // result of the StartStore Their DID operation
	key    *Future // DID construct where key is Future
	meta   *Future // Meta data stored to DID
	pw     *Future // Pairwise data stored to DID
	endp   *Future // When endpoint is started to fetch it's here

	sync.Mutex // when setting Future ptrs making sure that happens atomically

	pwMeta *PairwiseMeta // Meta data for pairwise

}

type PairwiseMeta struct {
	Name  string
	Route []string
}

type Pairwise struct {
	MyDID    string
	TheirDID string
	Meta     PairwiseMeta
}

func NewAgentDid(wallet managed.Wallet, f *Future) (ad *DID) {
	return &DID{wallet: wallet, data: f}
}

func NewDid(did, verkey string) (d *DID) {
	f := &Future{V: dto.Result{Data: dto.Data{Str1: did, Str2: verkey}}, On: Consumed}
	return &DID{data: f}
}

func NewOutDid(verkey string, route []string) (d *DID) {
	did := NewDid("", verkey)
	did.pwMeta = &PairwiseMeta{Route: route}
	return did
}

func NewDidWithKeyFuture(wallet managed.Wallet, did string, verkey *Future) (d *DID) {
	f := &Future{V: dto.Result{Data: dto.Data{Str1: did, Str2: ""}}, On: Consumed}
	d = &DID{wallet: wallet, data: f, key: verkey}
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
}

// Store stores this DID as their DID to given wallet. Work is done thru futures
// so the call doesn't block. The meta data is set "pairwise". See StoreResult()
// for status.
func (d *DID) Store(wallet int) {
	ds, vk, _ := d.data.Strs()

	idJSON := did.Did{Did: ds, VerKey: vk}
	json := dto.ToJSON(idJSON)
	f := new(Future)

	f.SetChan(did.StoreTheir(wallet, json))

	d.Lock()
	d.stored = f
	d.Unlock()

	go func() {
		defer err2.CatchTrace(func(err error) {}) // dont let crash on panics

		f := new(Future)
		f.SetChan(did.SetMeta(wallet, ds, "pairwise"))

		if f.Result().Err() == nil { // no error
			d.Lock()
			d.meta = f
			d.Unlock()
		}
	}()
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

func (d *DID) SavePairwiseForDID(wallet int, theirDID *DID, pw PairwiseMeta) {
	// check that DIDs are ready
	ok := d.data.Result().Err() == nil && theirDID.stored.Result().Err() == nil
	if ok {
		// encode to b64 string so that wrapper/indy does not attempt to decode json
		f := &Future{}
		meta := base64.StdEncoding.EncodeToString(dto.ToJSONBytes(pw))

		f.SetChan(pairwise.Create(wallet, theirDID.Did(), d.Did(), meta))
		theirDID.pw = f
	} else {
		glog.Error("Could not store pairwise info")
	}
}

func (d *DID) hasKeyData() bool {
	_, vk, _ := d.data.Strs()
	return vk != ""
}

func (d *DID) StartEndp(wallet int) {
	f := &Future{}
	f.SetChan(did.Endpoint(wallet, Pool(), d.Did()))
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
		V:  dto.Result{Data: dto.Data{Str1: ae.Endp, Str2: ae.Key}},
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
