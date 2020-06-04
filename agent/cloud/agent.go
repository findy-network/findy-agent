package cloud

import (
	"errors"
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pairwise"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sa"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/trans"
	"github.com/findy-network/findy-agent/agent/txp"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/findy-network/findy-wrapper-go/dto"
	indypw "github.com/findy-network/findy-wrapper-go/pairwise"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"golang.org/x/net/websocket"
)

type cnxMap map[string]*websocket.Conn

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
	Tr       txp.Trans        // Our transport layer for communication
	worker   *Agent           // worker agent to perform tasks when corresponding EA is not available
	ca       *Agent           // if this is worker agent (see prev) this is the CA
	callerPw *pairwise.Caller // the helper class for binding DID pairwise, this we are Caller
	calleePw *pairwise.Callee // the helper class for binding DID pairwise, this we are Callee
	cnxCh    chan bool        // Channel for triggering to shutdown the WebSocket communication
	cnxes    cnxMap           // web socket connections by DID
	cnxesLK  sync.Mutex       // web socket map's lock
	pwLock   sync.Mutex       // pw map lock, see below:
	pws      PipeMap          // Map of pairwise secure pipes by DID
	pwNames  PipeMap          // Map of pairwise secure pipes by name
}

type SeedAgent struct {
	RootDID string
	CADID   string
	*ssi.Wallet
}

func (s *SeedAgent) Prepare() (h comm.Handler, err error) {
	agent := &Agent{}
	agent.OpenWallet(*s.Wallet)

	rd := agent.LoadDID(s.RootDID)
	agent.SetRootDid(rd)
	if err := agent.LoadPwOnInit(); err != nil {
		glog.Error("cannot load pairwise for CA:", s.CADID)
		agent.CloseWallet()
		return nil, err
	}
	caDid := agent.Tr.PayloadPipe().In.Did()
	if caDid != s.CADID {
		glog.Warning("cloud agent DID is not correct")
	}
	return agent, nil
}

func NewSeedAgent(rootDid, caDid string, cfg *ssi.Wallet) *SeedAgent {
	return &SeedAgent{
		RootDID: rootDid,
		CADID:   caDid,
		Wallet:  cfg,
	}
}

func (a *Agent) SetCnxCh(cnxCh chan bool) {
	a.cnxCh = cnxCh
}

type PipeMap map[string]sec.Pipe

// NewEA creates a new EA without any initialization.
func NewEA() *Agent {
	return &Agent{DIDAgent: ssi.DIDAgent{Type: ssi.Edge}}
}

// NewTransportReadyEA creates a new EA and opens its wallet and inits its
// transport layer for CA communication.
func NewTransportReadyEA(walletCfg *ssi.Wallet) *Agent {
	ea := &Agent{DIDAgent: ssi.DIDAgent{Type: ssi.Edge}}
	ea.OpenWallet(*walletCfg)
	commPipe, _ := ea.PwPipe(pltype.HandshakePairwiseName)
	ea.Tr = trans.Transport{PLPipe: commPipe, MsgPipe: commPipe}
	return ea
}

// AttachAPIEndp sets the API endpoint for remove EA i.e. real EA not w-EA.
func (a *Agent) AttachAPIEndp(endp service.Addr) error {
	a.EAEndp = &endp
	theirDid := a.Tr.PayloadPipe().Out.Did()
	endpJSONStr := dto.ToJSON(endp)
	r := <-did.SetMeta(a.Wallet(), theirDid, endpJSONStr)
	return r.Err()
}

// AttachSAImpl sets implementation ID for SA to use for Mocks.
func (a *Agent) AttachSAImpl(implID string) {
	a.SAImplID = implID
}

// CallableEA tells if we can call EA from here, from Agency. It means that EA
// is offering a callable endpoint and we must have a pairwise do that.
func (a *Agent) CallableEA() bool {
	if !a.IsCA() {
		panic("not a CA")
	}
	if a.EAEndp == nil {
		theirDid := a.Tr.PayloadPipe().Out.Did()
		r := <-did.Meta(a.Wallet(), theirDid)
		if r.Err() == nil && r.Str1() != "pairwise" && r.Str1() != "" {
			a.EAEndp = new(service.Addr)
			dto.FromJSONStr(r.Str1(), a.EAEndp)
			glog.V(3).Info("got endpoint from META", r.Str1())
		}
	}
	attached := a.EAEndp != nil
	return attached
}

// CallEA makes a remove call for real EA and its API (issuer and verifier).
func (a *Agent) CallEA(plType string, im didcomm.Msg) (om didcomm.Msg, err error) {

	// An error in calling SA usually tells that SA is not present or a
	// network error did happened. It's better to let actual protocol
	// continue so other end will know. We cannot just interrupt the protocol.
	defer err2.Handle(&err, func() {
		glog.Error("Error in calling EA:", err)

		om = im            // rollback is the safest
		om.SetReady(false) // The way to tell that NO GO!!
		err = nil          // clear error so protocol continues NACK to other end
	})

	if a.CallableEA() {
		glog.V(3).Info("calling EA")
		return a.callEA(plType, im)
	} else if a.SAImplID != "" {
		glog.V(3).Info("call SA impl")
		return sa.Get(a.SAImplID)(plType, im)
	}
	// Default answer is definitely NO, we don't have issuer or prover
	om = im
	om.SetReady(false)
	info := "no SA endpoint or implementation ID set"
	glog.V(3).Info(info)
	om.SetInfo(info)
	return om, nil
}

// NotifyEA notifies the corresponding EA via web socket. Consider other options
// like apns, http, even rpc, etc.
func (a *Agent) NotifyEA(plType string, im didcomm.MessageHdr) {
	defer err2.CatchTrace(func(err error) {
		glog.Warning("cannot notify EA anymore:", err)
		glog.V(1).Info("---> cleaning up ws socked for this DID:", a.WDID())
		a.rmWs(a.WDID())
	})

	ws := a.cnx(a.WDID()) // Is there ongoing websocket..
	if ws == nil {        // .. live and well?
		return
	}

	creator := didcomm.CreatorGod.PayloadCreatorByType(plType)
	pl := creator.NewMsg(im.Thread().ID, plType, im)
	data := a.Tr.PayloadPipe().Encrypt(pl.JSON())
	err2.Check(websocket.Message.Send(ws, data))
}

func (a *Agent) callEA(plType string, msg didcomm.Msg) (om didcomm.Msg, err error) {
	defer err2.Return(&err)

	glog.V(5).Info("Calling EA:", a.EAEndp.Endp, plType)
	ipl, err := a.Tr.DIDComCallEndp(a.EAEndp.Endp, plType, msg)
	err2.Check(err)

	if ipl.Type() == pltype.ConnectionError {
		return om, fmt.Errorf("call ea api: %v", ipl.Message().Error())
	}

	return ipl.Message(), nil
}

func (a *Agent) Trans() txp.Trans {
	return a.Tr
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

func (a *Agent) Pw() pairwise.Saver {
	if a.calleePw != nil {
		return a.calleePw
	} else if a.callerPw != nil {
		return a.callerPw
	}
	return nil
}

// InOutPL handles messages of handshake protocol at the moment. Future will
// show if it stays or even expands. The messages processed here are the CA's.
// The EA end is handled by clients.
func (a *Agent) InOutPL(cnxAddr *endp.Addr, ipl didcomm.Payload) (opl didcomm.Payload, nonce string) {
	a.AssertWallet()

	glog.V(1).Info("Handle message type: " + ipl.Type())

	switch ipl.Type() {
	case pltype.ConnectionResponse: // clients 2nd message for us
		pw := a.callerPw                                    // continue with Pairwise
		im := pw.ReceiveResponse(ipl.Message().Encrypted()) // process it with input Payload
		if im.Nonce() != pw.Msg.Nonce() {
			glog.Error("Fatal error, EA/CA nonce mismatch")
		}

		endAddr, pwEndAddr, pwEndKey, name := "", "", "", ""

		// When we are a CA this is termination of the HANDSHAKE so we must
		// update the Transport layer to be able to communicate with all of
		// the EAs. Those who start communication sequences and those who
		// listen from ws.
		if a.IsCA() {
			meDID := a.LoadDID(cnxAddr.ReceiverDID())
			youDID := a.LoadDID(im.Did()) // remember set our EA and Transport layer
			cloudPipe := sec.Pipe{In: meDID, Out: youDID}
			transport := &trans.Transport{PLPipe: cloudPipe, MsgPipe: cloudPipe}
			a.Tr = transport
			endAddr = a.CAEndp(false).Address()
			workerEndpoint := a.CAEndp(true)
			pwEndAddr = workerEndpoint.Address()
			transport.Endp = pwEndAddr
			pwEndKey = workerEndpoint.VerKey
			name = a.callerPw.Pairwise.Endp
		} else {
			endAddr = a.BuildEndpURL()
		}
		nonce = pw.Msg.Nonce()

		pwEKey := service.Addr{Endp: pwEndAddr, Key: pwEndKey}
		initMsg := didcomm.MsgInit{
			Did:      im.Did(),
			VerKey:   im.VerKey(),
			Endpoint: endAddr,
			RcvrEndp: pwEKey,
			Name:     name,
		}

		opl = mesg.PayloadCreator.New(didcomm.PayloadInit{
			ID:      nonce,
			Type:    pltype.ConnectionAck,
			MsgInit: initMsg,
		})
		if a.IsCA() {
			agency.SaveRegistered()
		}

	// This arrives when on-boarding is started and this Cloud Agent is
	// allocated for that. Agency preprocess the message before this arrives
	// here. This is clients 1st message for us.
	case pltype.ConnectionHandshake:
		pw := pairwise.NewCallerPairwise(mesg.MsgCreator,
			a, a.RootDid(),
			pltype.ConnectionHandshake) // we start the pairwise sequence
		pw.Endp = ipl.Message().Endpoint().Endp // endpoint is used to transfer email/wallet name in handshake
		pw.Name = ipl.Message().Name()          // in this version the requested pairwise name is here
		pw.Build(true)                          // build Msg to be send
		nonce = pw.Msg.Nonce()                  // attach to Payload
		opl = mesg.PayloadCreator.NewMsg(nonce, pltype.ConnectionRequest, pw.Msg.(didcomm.Msg))
		a.callerPw = pw // save pairwise for client opl

	default:
		panic("we should not be here")
	}
	return opl, nonce
}

// ProcessPL is helper method to implement a CA client aka EA. This is not
// called by CA itself which is running on server. This for agents which are
// connecting Agency run CAs
func (a *Agent) ProcessPL(ipl didcomm.Payload) (opl didcomm.Payload, err error) {
	a.AssertWallet()

	p := comm.Packet{
		Payload:  ipl,
		Address:  nil,
		Receiver: a,
	}

	glog.V(1).Info("Handle message type: " + ipl.Type())

	switch ipl.Type() {
	// =============================================================
	// EA's response handlers to Payloads, note! EA == Client of CA
	// =============================================================
	case pltype.ConnectionRequest: // we receive from server as response from CA
		a.calleePw = pairwise.NewCalleePairwise(mesg.MsgCreator, a, ipl.Message())
		msg := a.calleePw.ConnReqToResp().(didcomm.Msg)
		opl = mesg.PayloadCreator.NewMsg(ipl.ID(), pltype.ConnectionResponse, msg)

	case pltype.ConnectionAck: // we receive from server as response from CA
		a.Pw().SaveEndpoint(ipl.Message().Endpoint().Endp)
		initMsg := didcomm.MsgInit{
			Info: "OP Tech Lab (c)",
		}
		opl = mesg.PayloadCreator.New(didcomm.PayloadInit{
			ID:      "0",
			Type:    pltype.ConnectionOk,
			MsgInit: initMsg,
		})

	default:
		opl = comm.ProcessMsg(p, func(im, om didcomm.Msg) (err error) {
			glog.Error("unknown payload")
			glog.Error(dto.ToJSON(im))
			return fmt.Errorf("unknown payload")
		})
	}
	return opl, nil
}

func (a *Agent) BuildEndpURL() (endAddr string) {
	hostname := utils.Settings.HostAddr()

	if a.IsCA() { // ===== this is HANDSHAKE
		cloudAgentDID := ""

		if a.Pw() != nil { // during and just after handshake for THIS CA
			cloudAgentDID = a.Pw().MeDID()
		} else if a.Tr.PayloadPipe().In != nil { // after Handshake we have Transport
			cloudAgentDID = a.Tr.PayloadPipe().In.Did()
		}

		endpAddr := endp.Addr{
			BasePath: hostname,
			Service:  utils.Settings.ServiceName(),
			PlRcvr:   cloudAgentDID,
			MsgRcvr:  cloudAgentDID,
			RcvrDID:  cloudAgentDID,
		}
		endAddr = endpAddr.Address()
	} else if a.IsEA() && a.Pw() != nil { // === This is a PAIRWISE CONN_OFFER
		var youDID string
		if a.Tr != nil && a.Tr.PayloadPipe().Out != nil { // if we Handshake is done we have Trans
			youDID = a.Tr.PayloadPipe().Out.Did() // outside access is always thru our CA
		} else {
			youDID = a.Pw().YouDID() // If this is ready
		}
		myCnxAddr := endp.Addr{
			BasePath: hostname,                     // This's what we know now
			Service:  utils.Settings.ServiceName(), // Set our agency's name to the address
			PlRcvr:   youDID,                       // Set our CA to a PL transfer DID
			MsgRcvr:  a.Pw().MeDID(),               // Set our PW DID to the actual msg receiver router
			RcvrDID:  a.Pw().MeDID(),               // Set our PW DID to the actual DID receiver
		}
		endAddr = myCnxAddr.Address()
	} else {
		// This should newer happen but this's easier for development than panic
		// we won't log this because we are here during the actual handshake or
		// pairwise
		endAddr = "TODO!"
	}
	return endAddr
}

// CAEndp returns endpoint of the CA or CA's w-EA's endp when wantWorker = true.
func (a *Agent) CAEndp(wantWorker bool) (endP *endp.Addr) {
	hostname := utils.Settings.HostAddr()
	if !a.IsCA() {
		return nil
	}
	caDID := a.Tr.PayloadPipe().In.Did()

	rcvrDID := caDID
	vk := a.Tr.PayloadPipe().In.VerKey()
	serviceName := utils.Settings.ServiceName()
	if wantWorker {
		rcvrDID = a.Tr.PayloadPipe().Out.Did()
		serviceName = utils.Settings.ServiceName2()
		// NOTE!! the VK is same 'because it's CA who decrypts invite PLs!
	}
	return &endp.Addr{
		BasePath: hostname,
		Service:  serviceName,
		PlRcvr:   caDID,
		MsgRcvr:  caDID,
		RcvrDID:  rcvrDID,
		VerKey:   vk,
	}
}

func (a *Agent) LoadPwOnInit() (err error) {
	defer err2.Return(&err)

	my, their := err2.StrStr.Try(a.Pairwise(pltype.HandshakePairwiseName))
	if my == "" { // legacy check, this can be removed later
		my, their = err2.StrStr.Try(a.Pairwise(pltype.ConnectionHandshake))
	}

	if my != "" {
		meDID := a.LoadDID(my)
		youDID := a.LoadDID(their)

		cloudPipe := sec.Pipe{In: meDID, Out: youDID}
		a.Tr = trans.Transport{PLPipe: cloudPipe, MsgPipe: cloudPipe}
		return nil
	}
	return errors.New("cannot find handshake")
}

func (a *Agent) AddWs(msgDID string, ws *websocket.Conn) {
	wsList := a.cnxMap()

	a.cnxesLK.Lock()
	defer a.cnxesLK.Unlock()
	wsList[msgDID] = ws
}

func (a *Agent) rmWs(msgDID string) {
	wsList := a.cnxMap()

	a.cnxesLK.Lock()
	defer a.cnxesLK.Unlock()
	delete(wsList, msgDID)
}

func (a *Agent) cnxMap() cnxMap {
	a.cnxesLK.Lock()
	defer a.cnxesLK.Unlock()
	if a.cnxes == nil {
		a.cnxes = make(cnxMap)
	}
	return a.cnxes
}

func (a *Agent) cnx(msgDID string) *websocket.Conn {
	wsList := a.cnxMap()

	a.cnxesLK.Lock()
	defer a.cnxesLK.Unlock()
	return wsList[msgDID]
}

func (a *Agent) PwPipe(pw string) (cp sec.Pipe, err error) {
	defer err2.Return(&err)

	a.pwLock.Lock()
	defer a.pwLock.Unlock()

	if secPipe, ok := a.pwNames[pw]; ok {
		return secPipe, nil
	}

	in, out, err := a.Pairwise(pw)
	err2.Check(err)

	if in == "" || out == "" {
		return cp, errors.New("cannot find pw")
	}

	cp.In = a.LoadDID(in)
	outDID := a.LoadDID(out)
	outDID.StartEndp(a.Wallet())
	cp.Out = outDID
	return cp, nil
}

// workerAgent creates worker agent for our EA if it isn't already done. Worker
// is a pseudo EA which presents EA in the could and so it is always ONLINE. By
// this other agents can connect to us even when all of our EAs are offline.
// This is under construction.
func (a *Agent) workerAgent(rcvrDID, suffix string) (wa *Agent) {
	if a.worker == nil {
		if rcvrDID != a.Tr.PayloadPipe().Out.Did() {
			glog.Error("Agent URL doesn't match with Transport")
			panic("Agent URL doesn't match with Transport")
		}
		aWallet := a.WalletH.Config().WorkerWalletBy(suffix)

		// getting wallet credentials
		// CA and EA wallets have same key, they have same root DID
		key, err := enclave.WalletKeyByDID(a.RootDid().Did())
		if err != nil {
			glog.Error("cannot get wallet key:", err)
			panic(err)
		}
		aWallet.Credentials.Key = key
		walletInitializedBefore := aWallet.Create()

		workerMeDID := a.LoadDID(rcvrDID)
		workerYouDID := a.Tr.PayloadPipe().In
		cloudPipe := sec.Pipe{In: workerMeDID, Out: workerYouDID}
		transport := trans.Transport{PLPipe: cloudPipe, MsgPipe: cloudPipe}
		glog.V(3).Info("Create worker transport: ", transport)

		a.worker = &Agent{
			DIDAgent: ssi.DIDAgent{
				Type:     ssi.Edge | ssi.Worker,
				Root:     a.RootDid(),
				DidCache: a.DidCache,
			},
			Tr:      transport, // Transport for EA is created here!
			ca:      a,
			pws:     make(PipeMap),
			pwNames: make(PipeMap),
		}
		a.worker.OpenWallet(*aWallet)
		// cleanup, secure enclave stuff, minimize time in memory
		aWallet.Credentials.Key = ""

		comm.ActiveRcvrs.Add(rcvrDID, a.worker)

		if !walletInitializedBefore {
			glog.V(2).Info("Creating a master secret into worker's wallet")
			masterSec, err := enclave.NewWalletMasterSecret(a.RootDid().Did())
			if err != nil {
				glog.Error(err)
				panic(err)
			}
			r := <-anoncreds.ProverCreateMasterSecret(a.worker.Wallet(), masterSec)
			if r.Err() != nil || masterSec != r.Str1() {
				glog.Error(r.Err())
				panic(r.Err())
			}
		}

		a.worker.loadPWMap()
	}
	return a.worker
}

func (a *Agent) MasterSecret() (string, error) {
	return enclave.WalletMasterSecretByDID(a.RootDid().Did())
}

// WDID returns DID string of the WEA.
func (a *Agent) WDID() string {
	return a.Tr.PayloadPipe().Out.Did()
}

// WEA returns CA's worker EA. It creates and inits it correctly if needed. The
// w-EA is a cloud allocated EA. Note! The TR is attached to worker EA here!
func (a *Agent) WEA() (wa *Agent) {
	if a.worker != nil {
		return a.worker
	}
	rcvrDID := a.Tr.PayloadPipe().Out.Did()
	return a.workerAgent(rcvrDID, "_worker")
}

func (a *Agent) WorkerEA() comm.Receiver {
	return a.WEA()
}

func (a *Agent) ExportWallet(key string, exportPath string) string {
	exportFile := exportPath
	url := exportPath
	if exportPath == "" {
		exportFile, url = utils.Settings.WalletExportPath(a.RootDid().Did())
	}
	exportCreds := wallet.Credentials{
		Path:                exportFile,
		Key:                 key,
		KeyDerivationMethod: "RAW",
	}
	a.Export.SetChan(wallet.Export(a.Wallet(), exportCreds))
	return url
}

func (a *Agent) loadPWMap() {
	a.AssertWallet()

	r := <-indypw.List(a.Wallet())
	if r.Err() != nil {
		glog.Error("ERROR: could not load pw map:", r.Err())
		return
	}

	pwd := indypw.NewData(r.Str1())

	a.pwLock.Lock()
	defer a.pwLock.Unlock()

	for _, d := range pwd {
		outDID := a.LoadDID(d.TheirDid)
		outDID.StartEndp(a.Wallet())
		p := sec.Pipe{
			In:  a.LoadDID(d.MyDid),
			Out: outDID,
		}

		a.pws[d.MyDid] = p
		a.pwNames[d.Metadata] = p
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
