package cloud

import (
	"errors"
	"sync"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

/*
Agent is the main abstraction of the package together with Agency. The agent
started as a CA but has been later added support for EAs and worker/cloud-EA as
well. This might be something we will change later. Agent's most important task
is/WAS to receive Payloads and process Messages inside them. And there are lots
of stuff to support that. That part of code is heavily under construction.

More concrete parts of the Agent are support for wallet, root DID, did cache.
Web socket connections are more like old relic, and that will change in future
for something else. It WAS part of the protocol STATE management.

Please be noted that Agent or more precisely CA is singleton by its nature per
EA it serves. So, Cloud Agent is a gateway to world for EA it serves. EAs are
mostly in mobile devices and handicapped by their nature. In our latest
architecture CA serves EA by creating a worker EA which lives in the cloud as
well. For now, in the most cases we have pair or agents serving each mobile EAs
here in the cloud: CA and w-EA.

There is Agent.Type where this Agent can be EA only. That type is used for test
and CLI Go clients.
*/
type Agent struct {
	ssi.DIDAgent

	// replace transport with caDID
	myDID *ssi.DID

	// worker agent performs the actual protocol tasks
	worker agentPtr

	// if this is worker agent (see worker field) this is the pointer to CA
	ca *Agent

	// all connections (pairwise) are cached by the Agent
	pwLock  sync.Mutex // pw map lock, see below:
	pws     PipeMap    // Map of pairwise secure pipes by DID
	pwNames PipeMap    // Map of pairwise secure pipes by name
}

type agentPtr struct {
	sync.RWMutex
	agent *Agent // private & named that you cannot use default dispatching
}

func (ap *agentPtr) testAndSet(set func() *Agent) *Agent {
	ap.Lock()
	defer ap.Unlock()

	if ap.agent == nil {
		ap.agent = set()
	}
	return ap.agent
}

func (ap *agentPtr) get() *Agent {
	ap.RLock()
	defer ap.RUnlock()

	return ap.agent
}

func (a *Agent) AutoPermission() bool {
	autoPermissionOn := a.SAImplID() == "permissive_sa"
	glog.V(1).Infof("auto permission = %v", autoPermissionOn)
	return autoPermissionOn
}

type SeedAgent struct {
	RootDID  string
	CADID    string
	CAVerKey string
	*ssi.Wallet
}

func (s *SeedAgent) Prepare() (h comm.Handler, err error) {
	agent := &Agent{myDID: ssi.NewDid(s.CADID, s.CAVerKey)}
	agent.OpenWallet(*s.Wallet)

	rd := agent.LoadDID(s.RootDID)
	agent.SetRootDid(rd)

	return agent, nil
}

func (s *SeedAgent) Migrate() (h comm.Handler, err error) {
	agent := &Agent{}
	agent.OpenWallet(*s.Wallet)

	rd := agent.LoadDID(s.RootDID)
	agent.SetRootDid(rd)

	caDID := agent.LoadDID(s.CADID)
	agent.SetMyDID(caDID)

	return agent, nil
}

func NewSeedAgent(rootDid, caDid, caVerKey string, cfg *ssi.Wallet) *SeedAgent {
	return &SeedAgent{
		RootDID:  rootDid,
		CADID:    caDid,
		CAVerKey: caVerKey,
		Wallet:   cfg,
	}
}

type PipeMap map[string]sec.Pipe

// NewEA creates a new EA without any initialization.
func NewEA() *Agent {
	return &Agent{DIDAgent: ssi.DIDAgent{Type: ssi.Edge}}
}

// AttachSAImpl sets implementation ID for SA to use for mocks and auto accepts.
func (a *Agent) AttachSAImpl(implID string) {
	defer err2.Catch(func(err error) {
		glog.Errorln("attach sa impl:", err)
	})
	a.SetSAImplID(implID)
	glog.V(3).Infof("setting implementation (%s)", a.SAImplID())
	if a.IsCA() {
		wa, ok := a.WorkerEA().(*Agent)
		if !ok {
			glog.Errorf("type assert, wrong agent type for %s",
				a.RootDid().Did())
		}
		wa.SetSAImplID(implID)
	}
}

func (a *Agent) SetMyDID(myDID *ssi.DID) {
	a.myDID = myDID
}

func (a *Agent) MyDID() *ssi.DID {
	return a.myDID
}

func (a *Agent) MyCA() comm.Receiver {
	if !a.IsWorker() {
		panic("not a worker agent! don't have CA")
	}
	if a.ca == nil {
		panic("CA is nil in Worker EA")
	}
	return a.ca
}

// CAEndp returns endpoint of the CA.
func (a *Agent) CAEndp() (endP *endp.Addr) {
	assert.D.True(a.IsCA())

	hostname := utils.Settings.HostAddr()
	caDID := a.MyDID().Did()
	rcvrDID := caDID
	vk := a.MyDID().VerKey()
	rcvrDID = a.WDID()
	serviceName := utils.Settings.ServiceName()

	return &endp.Addr{
		BasePath: hostname,
		Service:  serviceName,
		PlRcvr:   caDID,
		MsgRcvr:  caDID,
		RcvrDID:  rcvrDID,
		VerKey:   vk,
	}
}

func (a *Agent) PwPipe(pwName string) (cp sec.Pipe, err error) {
	defer err2.Return(&err)

	a.pwLock.Lock()
	defer a.pwLock.Unlock()

	if secPipe, ok := a.pwNames[pwName]; ok {
		return secPipe, nil
	}

	pw := try.To1(a.FindPWByName(pwName))

	if pw == nil || pw.MyDID == "" || pw.TheirDID == "" {
		return cp, errors.New("cannot find pw")
	}

	cp.In = a.LoadDID(pw.MyDID)
	outDID := a.LoadTheirDID(*pw)
	outDID.StartEndp(a.ManagedWallet(), pwName)
	cp.Out = outDID
	return cp, nil
}

// workerAgent creates worker agent for our EA if it isn't already done. Worker
// is a pseudo EA which presents EA in the cloud and so it is always ONLINE. By
// this other agents can connect to us even when all of our EAs are offline.
// This is under construction.
func (a *Agent) workerAgent(waDID, suffix string) (wa *Agent) {
	ca := a // to help us to read the code, receiver is CA
	return ca.worker.testAndSet(func() *Agent {
		glog.V(2).Infoln("starting worker agent creation process")
		assert.P.True(waDID == ca.WDID(), "Agent URL doesn't match with Transport")

		cfg := ca.WalletH.Config().(*ssi.Wallet)
		aWallet := cfg.WorkerWalletBy(suffix)

		// getting wallet credentials
		// CA and EA wallets have same key, they have same root DID
		key, err := enclave.WalletKeyByDID(ca.RootDid().Did())
		if err != nil {
			glog.Error("cannot get wallet key:", err)
			panic(err)
		}
		aWallet.Credentials.Key = key
		walletInitializedBefore := aWallet.Create()

		wca := &Agent{
			DIDAgent: ssi.DIDAgent{
				Type:     ssi.Edge | ssi.Worker,
				Root:     ca.RootDid(),
				DidCache: ca.DidCache.Clone(),
			},
			ca:      ca,
			pws:     make(PipeMap),
			pwNames: make(PipeMap),
			myDID:   ca.myDID,
		}

		wca.OpenWallet(*aWallet)
		// cleanup, secure enclave stuff, minimize time in memory
		aWallet.Credentials.Key = ""

		comm.ActiveRcvrs.Add(waDID, wca)

		if !walletInitializedBefore {
			glog.V(2).Info("Creating a master secret into worker's wallet")
			masterSec, err := enclave.NewWalletMasterSecret(ca.RootDid().Did())
			if err != nil {
				glog.Error(err)
				panic(err)
			}
			r := <-anoncreds.ProverCreateMasterSecret(wca.Wallet(), masterSec)
			if r.Err() != nil || masterSec != r.Str1() {
				glog.Error(r.Err())
				panic(r.Err())
			}
		}
		wca.loadPWMap()

		return wca
	})
}

func (a *Agent) ID() string {
	return a.WalletH.Config().ID()
}

func (a *Agent) MasterSecret() (string, error) {
	return enclave.WalletMasterSecretByDID(a.RootDid().Did())
}

// WDID returns DID string of the WA and CALLED from CA.
func (a *Agent) WDID() string {
	assert.D.True(a.IsCA())

	// in the gRPC API version both DIDs are same
	wDID := a.myDID.Did()

	return wDID
}

// WEA returns CA's worker agent. It creates and inits it correctly if needed.
// The TR is attached to worker EA here!
func (a *Agent) WEA() (wa *Agent) {
	ca := a
	if ca.worker.get() != nil {
		return ca.worker.get()
	}
	glog.V(4).Infoln("worker NOT ready, starting creation process")
	waDID := ca.WDID()
	return ca.workerAgent(waDID, "_worker")
}

func (a *Agent) WorkerEA() comm.Receiver {
	return a.WEA()
}

func (a *Agent) ExportWallet(key string, exportPath string) string {
	exportFile := exportPath
	fileLocation := exportPath
	if exportPath == "" {
		exportFile, fileLocation = utils.Settings.WalletExportPath(a.RootDid().Did())
	}
	exportCreds := wallet.Credentials{
		Path:                exportFile,
		Key:                 key,
		KeyDerivationMethod: "RAW",
	}
	a.Export.SetChan(wallet.Export(a.Wallet(), exportCreds))
	return fileLocation
}

func (a *Agent) loadPWMap() {
	defer err2.Catch(func(err error) {
		glog.Error("cannot load PW map:", err)
	})

	a.AssertWallet()

	connections, err := a.ManagedWallet().Storage().ConnectionStorage().ListConnections()
	err2.Check(err)

	a.pwLock.Lock()
	defer a.pwLock.Unlock()

	for _, conn := range connections {
		outDID := a.LoadTheirDID(conn)
		outDID.StartEndp(a.ManagedWallet(), conn.ID)
		p := sec.Pipe{
			In:  a.LoadDID(conn.MyDID),
			Out: outDID,
		}

		a.pws[conn.MyDID] = p
		a.pwNames[conn.ID] = p
	}
}

func (a *Agent) AddToPWMap(me, you *ssi.DID, name string) sec.Pipe {
	pipe := sec.Pipe{
		In:  me,
		Out: you,
	}

	a.pwLock.Lock()
	defer a.pwLock.Unlock()

	a.pws[me.Did()] = pipe
	a.pwNames[name] = pipe

	return pipe
}

func (a *Agent) AddPipeToPWMap(p sec.Pipe, name string) {
	a.pwLock.Lock()
	defer a.pwLock.Unlock()

	a.pws[p.In.Did()] = p
	a.pwNames[name] = p
}

func (a *Agent) SecPipe(meDID string) sec.Pipe {
	a.pwLock.Lock()
	defer a.pwLock.Unlock()

	return a.pws[meDID]
}
