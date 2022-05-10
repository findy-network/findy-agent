package comm

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
	storage "github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/core"
)

type Receiver interface {
	MyDID() core.DID
	MyCA() Receiver
	WorkerEA() Receiver
	ExportWallet(key string, exportPath string) (url string)
	RootDid() core.DID
	SendNYM(targetDid *ssi.DID, submitterDid, alias, role string) (err error)
	LoadDID(did string) core.DID
	LoadTheirDID(connection storage.Connection) core.DID
	WDID() string
	PwPipe(pw string) (cp sec.Pipe, err error)
	NewOutDID(didStr string, verKey ...string) (id core.DID, err error)
	Wallet() int
	ManagedWallet() (managed.Wallet, managed.Wallet)
	Pool() int
	FindPWByID(id string) (pw *storage.Connection, err error)
	AttachSAImpl(implID string)
	AddToPWMap(me, you core.DID, name string) sec.Pipe
	SaveTheirDID(did, vk string) (err error)
	CAEndp(connID string) (endP *endp.Addr)
	AddPipeToPWMap(p sec.Pipe, name string)
	MasterSecret() (string, error)
	AutoPermission() bool
	ID() string
}

type Receivers struct {
	Rcvrs map[string]Receiver
	Lk    sync.Mutex
}

var ActiveRcvrs = Receivers{
	Rcvrs: make(map[string]Receiver),
}

func (rs *Receivers) Add(DID string, r Receiver) {
	rs.Lk.Lock()
	defer rs.Lk.Unlock()
	rs.Rcvrs[DID] = r
}

func (rs *Receivers) Get(DID string) Receiver {
	rs.Lk.Lock()
	defer rs.Lk.Unlock()
	return rs.Rcvrs[DID]
}

// Handler can be Agency or Agent. They can input Payloads.
type Handler interface {
	// TODO: lapi, should we consider something else for handler after
	// refactoring
	//InOutPL(addr *endp.Addr, payload didcomm.Payload) (response didcomm.Payload, nonce string)
}

// SeedHandler is preloaded cloud agent which is not initialized yet.
type SeedHandler interface {
	Prepare() (Handler, error)
}
