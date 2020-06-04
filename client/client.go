/*
Package client implements an agency client, an Edge Agent. It is used for for
integration tests, where it allows run client and server code in same process
which helps for example debugging a lot.

Similar functionality can be found from cmds package which is preferred way to
build CLI. Please see findy-cli for more information.

Please note, that this client implementation is interim. For that reason the
code is not cleaned up or streamlined. When we decide if this API is permanent,
refactoring will proceed.
*/
package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/agency"
	"github.com/optechlab/findy-agent/agent/cloud"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/e2"
	"github.com/optechlab/findy-agent/agent/endp"
	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/prot"
	"github.com/optechlab/findy-agent/agent/service"
	"github.com/optechlab/findy-agent/agent/ssi"
	trans2 "github.com/optechlab/findy-agent/agent/trans"
	"github.com/optechlab/findy-agent/agent/utils"
	"github.com/optechlab/findy-agent/cmds"
	"github.com/optechlab/findy-agent/cmds/onboard"
	didexchange "github.com/optechlab/findy-agent/std/didexchange/invitation"
	"github.com/optechlab/findy-go/dto"
	"github.com/optechlab/findy-go/pairwise"
	"github.com/optechlab/findy-go/wallet"
)

// Client is a helper struct which implements an EA. The EA can be used to
// connect agency and perform CA API calls to EA's CA's. Note that all of the
// protocol calls are async. They return task ID which can be used to query how
// task is proceeded.
type Client struct {
	Email       string       // Email address used for handshake/onboarding
	BaseAddress string       // URL for agency which serves this client (EA)
	Wallet      *ssi.Wallet  // EA's local wallet
	agent       *cloud.Agent // Our EA
	setAgent    bool         // tells if agent is already set
}

// SetAgent sets the agent from outside so it will be reused not created.
func (edge *Client) SetAgent(a *cloud.Agent) {
	edge.setAgent = true
	edge.agent = a
}

// Listen is a helper function to be used from services to be able to listen web
// socket to receive realtime notifications from CA. It needs a callback as an
// argument. Currently it's called from SA implementations who use us as a
// framework.
func (edge *Client) Listen(f trans2.EchoListener) (err error) {
	defer err2.Annotate("web socket listen", &err)

	if edge.agent == nil || !edge.setAgent {
		edge.agent = cloud.NewEA()
		edge.agent.OpenWallet(*edge.Wallet)
		defer func() {
			edge.agent.CloseWallet()
			edge.agent = nil
		}()
	}

	cloudPipe := e2.Pipe.Try(edge.agent.PwPipe(pltype.HandshakePairwiseName))
	edgePipe := cloudPipe
	trans := trans2.Transport{PLPipe: cloudPipe, MsgPipe: edgePipe}
	edge.agent.Tr = trans

	log.Println("Start web socket loop")
	err2.Check(trans.WsListenLoop("ca-apiws", f))

	return nil
}

// IsInit returns if the client is initialized properly.
func (edge *Client) IsInit() bool {
	return edge.Wallet != nil && edge.BaseAddress != ""
}

// Handshake is a function to on-board the EA to this Agency. It creates the
// client side wallet and implements EA's part of the pairwise protocol. The
// currently implemented protocol is a version of old indy a2a protocol, which
// was a precursor of the Aries connection/didexhange protocol.
// todo: we move this later, this is here for tests at the moment
func (edge *Client) Handshake() (theirDid string,
	finalPL *mesg.Payload, eaEnp service.Addr, err error) {

	r, err := onboard.Cmd{
		Cmd: cmds.Cmd{
			WalletName: edge.Wallet.Config.ID,
			WalletKey:  edge.Wallet.Credentials.Key,
		},
		Email:      edge.Email,
		AgencyAddr: edge.BaseAddress,
	}.Exec(os.Stdout)
	return r.CADID, nil, r.ServiceAddr, err
}

// EndpointInfo is helper struct for ExportLocalWallet.
type EndpointInfo struct {
	CAEndpoint string       `json:"caEndpoint"`
	Pairwise   service.Addr `json:"pairwise"`
}

// ExportLocalWallet exports local wallet.
func (edge *Client) ExportLocalWallet(filePath, exportKey,
	theirDid string, handshakePayload *mesg.Payload) (err error) {

	defer err2.Annotate("ExportLocalWallet", &err)

	edge.agent = cloud.NewEA()
	edge.agent.OpenWallet(*edge.Wallet)
	defer edge.agent.CloseWallet()

	if handshakePayload != nil {
		// TODO: add version number
		info := &EndpointInfo{
			CAEndpoint: handshakePayload.Message.Endpoint,
			Pairwise: service.Addr{
				Endp: handshakePayload.Message.RcvrEndp,
				Key:  handshakePayload.Message.RcvrKey,
			},
		}
		infoJSON := err2.Bytes.Try(json.Marshal(info))
		infoJSONStr := string(infoJSON)
		log.Println("Setting meta", infoJSONStr, "for DID", theirDid)
		r := <-pairwise.SetMeta(edge.agent.Wallet(), theirDid, infoJSONStr)
		err2.Check(r.Err())
	}

	edge.agent.ExportWallet(exportKey, filePath)
	err2.Check(edge.agent.Export.Result().Err())

	return nil
}

// ServicePing pings the agency. It's used to check if an agency is running and
// well.
func (edge *Client) ServicePing() (err error) {
	p := mesg.Payload{}

	endpointAdd := &endp.Addr{
		BasePath: edge.BaseAddress,
		Service:  agency.APIPath,
		PlRcvr:   "ping",
	}

	payload, err := sendAndWaitPayload(&p, endpointAdd, 0)
	if err != nil {
		return fmt.Errorf("ping: %s", err)
	}

	log.Println("ping response: ", payload)
	return
}

// PingCA pings EA's CA.
func (edge *Client) PingCA() (err error) {
	edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
	defer edge.agent.CloseWallet()

	tr := edge.agent.Trans()

	m := &mesg.Msg{Info: "Pinging from here..."}
	ipl, err := tr.Call(pltype.CAPingOwnCA, m)
	if err != nil {
		return err
	}
	fmt.Println("Endpoint from the server:")
	fmt.Println(ipl.Message.Endpoint)
	ep := endp.NewClientAddr(ipl.Message.Endpoint)
	//ep.Service
	ep.RcvrDID = tr.MessagePipe().In.Did()
	fmt.Println("My endpoint and key:")
	fmt.Println(ep.Address(), edge.agent.Tr.PayloadPipe().Out.VerKey())
	return err
}

// GetWallet prepares worker EA's (cloud allocated agent) wallet for download
// and returns the download URL.
func (edge *Client) GetWallet() (name string, err error) {
	defer err2.Annotate("get wallet", &err)
	edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
	defer edge.agent.CloseWallet()

	tr := edge.agent.Trans()

	r := <-wallet.GenerateKey("")
	err = r.Err()
	err2.Check(err)
	walletKey := r.Str1()
	m := &mesg.Msg{VerKey: walletKey}
	ipl := e2.Payload.Try(tr.Call(pltype.CAWalletGet, m))
	walletURL := ipl.Message.Endpoint
	if walletURL == "" {
		return "", errors.New("endpoint empty")
	}
	// we could put this to go func together with the rest of the block.
	exportPath := os.Getenv("TEST_WORKDIR")
	if len(exportPath) == 0 {
		currentUser, e := user.Current()
		err2.Check(e)
		exportPath = currentUser.HomeDir
	}
	exportPath = filepath.Join(exportPath, "wallets")
	walletFile := err2.String.Try(comm.FileDownload(exportPath, "", walletURL))
	log.Println(walletURL)
	newWallet := edge.Wallet.WorkerWallet()
	impCfg := wallet.Credentials{Path: walletFile, Key: walletKey}

	// we could just return the wallet name but we still should make sure that
	// the main process won't die before import is ready! So, let's wait.
	r = <-wallet.Import(newWallet.Config, newWallet.Credentials, impCfg)
	err = r.Err()
	err2.Check(err)
	return newWallet.Config.ID, nil
}

// CreateSchema calls create schema API from agency which writes it to ledger as
// well.
func (edge *Client) CreateSchema(schema *ssi.Schema) (schID string, err error) {
	edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
	defer edge.agent.CloseWallet()

	trans := edge.agent.Trans()

	m := &mesg.Msg{Schema: schema}

	ipl, err := trans.Call(pltype.CASchemaCreate, m)
	if err != nil {
		return "", fmt.Errorf("create schema: %s", err)
	}
	if ipl.Type == pltype.ConnectionError {
		return "", fmt.Errorf("create schema: %v", ipl.Message.Error)
	}
	schID = ipl.Message.Schema.ID
	return schID, nil
}

// CreateCredDef calls agency to create a cred def and write it to the ledger.
func (edge *Client) CreateCredDef(schID string, tag string) (credDefID string, err error) {
	edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
	defer edge.agent.CloseWallet()

	trans := edge.agent.Trans()

	m := &mesg.Msg{Schema: &ssi.Schema{ID: schID}, Info: tag}

	ipl, err := trans.Call(pltype.CACredDefCreate, m)

	if err != nil {
		return "", fmt.Errorf("create creddef: %s", err)
	}
	if ipl.Type == pltype.ConnectionError {
		return "", fmt.Errorf("create creddef: %v", ipl.Message.Error)
	}
	credDefID = ipl.Message.ID
	return credDefID, nil
}

// CreateDID calls agency to create DID and write it to the ledger.
func (edge *Client) CreateDID() (DID string, err error) {
	edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
	defer edge.agent.CloseWallet()

	trans := edge.agent.Trans()

	newDID := edge.agent.CreateDID("")
	originalNonce := utils.NewNonceStr()
	msg := &mesg.Msg{
		Nonce:  originalNonce,
		Did:    newDID.Did(),
		VerKey: newDID.VerKey(),
	}

	ipl, err := trans.Call(pltype.CALedgerWriteDid, msg)
	if err != nil {
		return "", fmt.Errorf("pairwise: %s", err)
	}
	if ipl.Type == pltype.ConnectionError {
		return "", fmt.Errorf("pairwise: %v", ipl.Message.Error)
	}
	return newDID.Did(), nil
}

// GetCredDef calls agency to return cred def from the ledger.
func (edge *Client) GetCredDef(credDefID string) (err error) {
	edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
	defer edge.agent.CloseWallet()

	trans := edge.agent.Trans()

	originalNonce := utils.NewNonceStr()

	msg := &mesg.Msg{
		Nonce: originalNonce,
		ID:    credDefID,
	}

	ipl, err := trans.Call(pltype.CALedgerGetCredDef, msg)
	if err != nil {
		return fmt.Errorf("submit: %s", err)
	}
	if ipl.Type == pltype.ConnectionError {
		return fmt.Errorf("submit: %v", ipl.Message.Error)
	}
	log.Println(ipl.Message.ID)
	log.Println(dto.ToJSON(ipl.Message.Msg))
	return nil
}

// GetCredDef calls agency to get schema from ledger.
func (edge *Client) GetSchema(schemaID string) (err error) {
	edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
	defer edge.agent.CloseWallet()

	trans := edge.agent.Trans()

	originalNonce := utils.NewNonceStr()

	msg := &mesg.Msg{
		Nonce: originalNonce,
		ID:    schemaID,
	}

	ipl, err := trans.Call(pltype.CALedgerGetSchema, msg)
	if err != nil {
		return fmt.Errorf("submit: %s", err)
	}
	if ipl.Type == pltype.ConnectionError {
		return fmt.Errorf("submit: %v", ipl.Message.Error)
	}
	log.Println("schema ID:")
	log.Println(ipl.Message.ID)
	//log.Println("schema JSON:")
	//log.Println(dto.ToJSON(ipl.Message.Msg))
	return nil
}

// CreatePW calls agency to create pairwise to other agent.
func (edge *Client) CreatePW(endpoint, endpKey, edgePw string) (tID string, err error) {
	defer err2.Annotate("create pw", &err)

	om := &mesg.Msg{
		Info: edgePw,
		Invitation: &didexchange.Invitation{
			ID:              edgePw,
			Type:            pltype.AriesConnectionInvitation,
			ServiceEndpoint: endpoint,
			RecipientKeys:   []string{endpKey},
			Label:           edgePw,
		},
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAPairwiseCreate, om))
	log.Println("Task #: ", im.ID)
	return im.ID, nil
}

// PwFromInvitation calls agency to create pairwise to other agent.
func (edge *Client) PwFromInvitation(i *didexchange.Invitation) (tID string, err error) {
	defer err2.Annotate("pw from invitation", &err)

	om := &mesg.Msg{
		Info:       i.Label,
		Invitation: i,
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAPairwiseCreate, om))
	log.Println("Task #: ", im.ID)
	return im.ID, nil
}

// TrustPingPW calls agency to create pairwise and end it with trust ping.
func (edge *Client) TrustPingPW(edgePw string) (tID string, err error) {
	defer err2.Annotate("trust ping", &err)
	om := &mesg.Msg{
		Name: edgePw,
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CATrustPing, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// JWT is not implemented yet.
func (edge *Client) JWT() (jwt string, err error) {
	defer err2.Annotate("requesting jwt", &err)

	om := &mesg.Msg{}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAGetJWT, om))
	log.Println("ok jwt request")
	return im.ID, nil
}

// CredReq asks agency to start issuing protocol with propose i.e. from holder's
// side.
func (edge *Client) CredReq(
	edgePw, credDefID,
	email, verID string) (tID string, err error) {

	defer err2.Annotate("cred req", &err)

	emailCred := didcomm.CredentialAttribute{Name: "email", Value: email}

	om := &mesg.Msg{
		Name:            edgePw,
		CredDefID:       &credDefID,
		CredentialAttrs: &[]didcomm.CredentialAttribute{emailCred},
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CACredRequest, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// CredOffer asks agency to start issuing protocol with offer i.e. from issuer's
// side.
func (edge *Client) CredOffer(
	edgePw, credDefID, email string) (tID string, err error) {

	om := &mesg.Msg{
		Name:      edgePw,
		CredDefID: &credDefID,
		CredentialAttrs: &[]didcomm.CredentialAttribute{
			{Name: "email", Value: email},
		},
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CACredOffer, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// PwAndCredReq asks agency to perform two protocols in a row. 1st make
// pairwise, 2nd start issuing protocol with propose, all from holder's side.
func (edge *Client) PwAndCredReq(
	endpoint, endpKey, edgePw, credDefID,
	email, veriID string) (tID string, err error) {

	defer err2.Annotate("pw and cred req", &err)

	emailCred := didcomm.CredentialAttribute{Name: "email", Value: email}
	om := &mesg.Msg{
		Info: edgePw,
		Invitation: &didexchange.Invitation{
			ID:              edgePw,
			Type:            pltype.AriesConnectionInvitation,
			ServiceEndpoint: endpoint,
			RecipientKeys:   []string{endpKey},
			Label:           edgePw,
		},
		CredDefID:       &credDefID,
		CredentialAttrs: &[]didcomm.CredentialAttribute{emailCred},
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAPairwiseAndCredReq, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// PwAndProofProp asks agency to perform two protocols in a row. 1st make
// pairwise, 2nd start proof protocol with propose, all from holder/prover side.
func (edge *Client) PwAndProofProp(
	endpoint, endpKey, edgePw, credDefID, email string) (tID string, err error) {
	defer err2.Annotate("pw and proof prop", &err)

	om := &mesg.Msg{
		Info: edgePw,
		Invitation: &didexchange.Invitation{
			ID:              edgePw,
			Type:            pltype.AriesConnectionInvitation,
			ServiceEndpoint: endpoint,
			RecipientKeys:   []string{endpKey},
			Label:           edgePw,
		},
		CredDefID: &credDefID,
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAPairwiseAndProofProp, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// PwAndTrustPing asks agency to perform two protocols in a row. 1st make
// pairwise, 2nd make a trust ping.
func (edge *Client) PwAndTrustPing(
	endpoint, endpKey, edgePw string) (tID string, err error) {
	defer err2.Annotate("pw and trust ping", &err)

	om := &mesg.Msg{
		Info: edgePw,
		Invitation: &didexchange.Invitation{
			ID:              edgePw,
			Type:            pltype.AriesConnectionInvitation,
			ServiceEndpoint: endpoint,
			RecipientKeys:   []string{endpKey},
			Label:           edgePw,
		},
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAPairwiseAndTrustPing, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// ProofProp asks agency to start proof protocol with propose proof from
// prover's side.
func (edge *Client) ProofProp(edgePw, email string) (tID string, err error) {
	defer err2.Annotate("proof prop", &err)
	om := &mesg.Msg{
		Name: edgePw,
		Info: email, // this is Values field in PresentProofRep
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAProofPropose, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// ProofRequest asks agency to start proof protocol with proof request from
// verifier's sid.
func (edge *Client) ProofRequest(edgePw, credDefID string) (tID string, err error) {
	defer err2.Annotate("proof req", &err)

	om := &mesg.Msg{
		Name: edgePw, // PW name to whom to send the Proof Req
		ProofAttrs: &[]didcomm.ProofAttribute{
			{
				Name:      "email",
				CredDefID: credDefID,
			},
		},
	}

	im := e2.Msg.Try(edge.MsgToCA(pltype.CAProofRequest, om))
	log.Println("============== Task #: ", im.ID, om.Nonce)
	return im.ID, nil
}

// SendMsg asks agency to send basic message to other agent defined by pairwise.
func (edge *Client) SendMsg(edgePw, ID, msg string) (tID string, err error) {
	defer err2.Annotate("proof prop", &err)

	om := &mesg.Msg{
		Name: edgePw,
		ID:   ID,
		Info: msg,
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CABasicMessage, om))
	log.Println("============== Task #: ", im.ID)
	return im.ID, nil
}

// TaskReady calls agency to get task status.
func (edge *Client) TaskReady(taskID string) (yes bool, err error) {
	defer err2.Annotate("task ready", &err)

	// Use task status API here to trigger and
	// test the protocol status setters
	im := e2.Msg.Try(edge.MsgToCA(
		pltype.CATaskStatus,
		&mesg.Msg{ID: taskID},
	))
	ready := false
	if im.Error != "" {
		err = errors.New(im.Error)
	} else {
		if body, ok := im.Body.(map[string]interface{}); ok {
			if taskStatus, ok := body["status"].(string); ok {
				ready = taskStatus == prot.StatusReady
				if ready {
					// Ensure that there is always status payload
					if _, ok := body["payload"].(map[string]interface{}); !ok {
						// TODO: handle this properly
						panic("Task is ready, but no payload!")
					}
				}
			}
		}
	}
	log.Println("Task", taskID, "ready: ", ready)
	return ready, err
}

// TaskStatus is a Client function just for to send CA API message as a test.
// Note that this function is only intended to be used as part of the
// integration tests.
func (edge *Client) TaskStatus(taskID string) (err error) {
	defer err2.Annotate("task status", &err)

	_ = e2.Msg.Try(edge.MsgToCA(pltype.CATaskStatus, &mesg.Msg{ID: taskID}))

	return nil
}

// ContinueProtocol calls agency to tell a certain protocol (taskID) does user
// accept to continue proofing protocol.
func (edge *Client) ContinueProtocol(taskID string, allow bool) (err error) {
	defer err2.Annotate("continue protocol", &err)

	om := &mesg.Msg{
		ID:    taskID,
		Ready: allow,
	}
	err2.Empty.Try(edge.MsgToCA(pltype.CAContinuePresentProofProtocol, om))

	return nil
}

// ContinueIssuingProtocol calls agency to tell a certain protocol (taskID) does
// user accept to continue issuing protocol.
func (edge *Client) ContinueIssuingProtocol(taskID string, allow bool) (err error) {
	defer err2.Annotate("continue issuing protocol", &err)

	om := &mesg.Msg{
		ID:    taskID,
		Ready: allow,
	}
	err2.Empty.Try(edge.MsgToCA(pltype.CAContinueIssueCredentialProtocol, om))

	return nil
}

// SetSAImpl calls agency to set EA's (service agent) current SA implementation.
// Please note this is more useful for integration tests. See SetAPIEndg for
// real use.
func (edge *Client) SetSAImpl(ImplID string) (err error) {
	defer err2.Annotate("set sa impl", &err)
	om := &mesg.Msg{
		ID:    ImplID,
		Ready: true, // we want notifications about tasks
	}
	err2.Empty.Try(edge.MsgToCA(pltype.CAAttachEADefImpl, om))
	return nil
}

// SetAPIEndp calls an agency to set service API endpoint for an agent.
func (edge *Client) SetAPIEndp(endp service.Addr) (err error) {
	defer err2.Annotate("set sa endp", &err)
	om := &mesg.Msg{
		RcvrEndp: endp.Endp,
		RcvrKey:  endp.Key,
		Ready:    true, // we want notifications about tasks to this endoint as well
	}
	err2.Empty.Try(edge.MsgToCA(pltype.CAAttachAPIEndp, om))
	return nil
}

// PingAPIEndp asks an agency to ping service API endpoint.
func (edge *Client) PingAPIEndp() (err error) {
	defer err2.Annotate("ping sa endp", &err)
	om := &mesg.Msg{
		Info: "Here we are",
	}
	im := e2.Msg.Try(edge.MsgToCA(pltype.CAPingAPIEndp, om))
	log.Println(im.Info)
	if im.Ready {
		log.Println("sa ping OK")
	} else {
		log.Println("sa ping ERROR")
	}
	return nil
}

// MsgToCA send what ever message to an agency.
func (edge *Client) MsgToCA(t string, msg *mesg.Msg) (in mesg.Msg, err error) {
	if edge.agent == nil || !edge.setAgent {
		edge.agent = cloud.NewTransportReadyEA(edge.Wallet)
		defer func() {
			edge.agent.CloseWallet()
			edge.agent = nil
		}()
	}

	trans := edge.agent.Trans()

	ipl, err := trans.Call(t, msg)
	if err != nil {
		return mesg.Msg{}, fmt.Errorf("submit: %s", err)
	}
	if ipl.Type == pltype.ConnectionError {
		return mesg.Msg{}, fmt.Errorf("submit: %v", ipl.Message.Error)
	}
	return ipl.Message, nil
}

func postRequest(urlStr string, msg io.Reader, timeout time.Duration) (p *mesg.Payload, err error) {
	data, err := comm.SendAndWaitReq(urlStr, msg, timeout)
	if err != nil {
		return nil, fmt.Errorf("reading body: %s", err)
	}
	p = mesg.NewPayload(data)
	if p.Message.Error != "" {
		err = fmt.Errorf("http POST response: %s", p.Message.Error)
	}

	return
}

func sendAndWaitPayload(p *mesg.Payload, endpoint *endp.Addr, nonce uint64) (rp *mesg.Payload, err error) {
	// BLOCKING CALL to make endpoint request this time, proper for handshakes
	return postRequest(endpoint.Address(), bytes.NewReader(p.JSON()), utils.Settings.Timeout())
}
