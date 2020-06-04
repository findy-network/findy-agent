package comm

import (
	"sync"

	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/endp"
	"github.com/optechlab/findy-agent/agent/sec"
	"github.com/optechlab/findy-agent/agent/service"
	"github.com/optechlab/findy-agent/agent/ssi"
	"github.com/optechlab/findy-agent/agent/txp"
	"golang.org/x/net/websocket"
)

type Receiver interface {
	Trans() txp.Trans
	MyCA() Receiver
	WorkerEA() Receiver
	ExportWallet(key string, exportPath string) (url string)
	BuildEndpURL() (endAddr string)
	RootDid() *ssi.DID
	SendNYM(targetDid *ssi.DID, submitterDid, alias, role string) (err error)
	LoadDID(did string) *ssi.DID
	WDID() string
	PwPipe(pw string) (cp sec.Pipe, err error)
	Wallet() int
	Pool() int
	FindPW(my string) (their string, pwname string, err error)
	CallEA(plType string, im didcomm.Msg) (om didcomm.Msg, err error)
	NotifyEA(plType string, im didcomm.MessageHdr)
	AttachAPIEndp(endp service.Addr) error
	CallableEA() bool
	AttachSAImpl(implID string)
	AddToPWMap(me, you *ssi.DID, name string) sec.Pipe
	SaveTheirDID(did, vk string, writeNYM bool) (err error)
	CAEndp(wantWorker bool) (endP *endp.Addr)
	AddPipeToPWMap(p sec.Pipe, name string)
	AddWs(msgDID string, ws *websocket.Conn)
	SetCnxCh(cnxCh chan bool)
	MasterSecret() (string, error)
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
	InOutPL(addr *endp.Addr, payload didcomm.Payload) (response didcomm.Payload, nonce string)
}

// SeedHandler is preloaded cloud agent which is not initialized yet.
type SeedHandler interface {
	Prepare() (Handler, error)
}
