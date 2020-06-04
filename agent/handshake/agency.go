/*
Package handshake is abstraction for agency level handshake protocol which
on-boards new clients to the agency. These clients are called EAs (Edge Agents).
Each EA has one CA to present it on the network. User can have many EAs which
all are connected to one CA.
*/
package handshake

import (
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pairwise"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

var Hub *Agency

func init() {
	Hub = new(Agency)
	agency.SetAgencyHandler(Hub)
}

// These are the named endpoints for pre-cases which are handled without pre-encryption.
// Both, these constants and below table endpoints() are needed.
const (
	PingHandlerEndpoint = "ping"
	HandlerEndpoint     = "handshake"
)

// Server's error messages
const (
	AlreadyExistError = "UNKNOWN_TYPE" // Going to production so we don't tell this
	UnknownTypeError  = "UNKNOWN_TYPE"
)

var steward *cloud.Agent // Steward Agent, which allows us on-board Agents for Edge Clients

/*
Agency is a singleton to encapsulate agency's settings data. It is the master
class for all of the services offered by the agency. It includes registers for
service handlers, aka endpoint handlers*. Most of the services are forwarded to
CAs and just few of them are offered by the agency it self, like on-boarding.

The Agency is still in heavy construction. There are some fields which are
obsolete or not used. The code base is not clean yet.

The Agency holds DID for the steward which it gets from the startup. It keeps
track for CAs (handlers) and their wallets (register).

*Endpoint handler was very early concept of the development, when all of the
endpoints, both agency endpoints and CA endpoints were routed by this agency
and which where served by same URL root. The selection between agency endpoints
and CA endpoints were done by reserved keywords for agency endpoints and DID
values for CA endpoints. NOW the routing is done by URL paths where and /api/
endpoints are routed to Agency.APICall()

Please see server.go for more information about service endpoints.
*/
type Agency struct{}

// Builds new trust anchor agent and its wallet. This is quite slow process. In
// future we could build them in advance to pool where we could allocate them
// when needed. Needs to wallet renaming or indexing.
func anchorAgent(email string) (agent *cloud.Agent, err error) {
	defer err2.Annotate("create archor", &err)

	key := err2.String.Try(enclave.NewWalletKey(email))
	defer func() {
		key = ""
	}()
	rippedEmail := strings.Replace(email, "@", "_", -1)

	// Build new agent with wallet
	agent = &cloud.Agent{}
	name := rippedEmail
	aw := ssi.NewRawWalletCfg(name, key)
	aw.Create()
	agent.OpenWallet(*aw)

	// Bind pairwise Steward <-> New Agent
	rootDid := steward.RootDid()
	caller := pairwise.NewCallerPairwise(mesg.MsgCreator, steward, rootDid,
		pltype.ConnectionTrustAgent)
	caller.Build(false)
	callee := pairwise.NewCalleePairwise(mesg.MsgCreator, agent, caller.Msg)
	responseMsg := callee.RespMsgAndOurDID()
	caller.Bind(responseMsg)

	// Promote new agent by Trusted Anchor DID
	anchorDid := agent.CreateDID("")
	err2.Check(steward.SendNYM(anchorDid, rootDid.Did(),
		findy.NullString, "TRUST_ANCHOR"))

	// Use the anchor DID as a submitter/root DID to Ledger
	agent.SetRootDid(anchorDid)

	err2.Check(enclave.SetKeysDID(key, anchorDid.Did()))

	return agent, nil
}

// InOutPL is Handler implementation which means that Agency can serve certain
// APIs. Most of the other HTTP call are routed to CAs.
func (a *Agency) InOutPL(
	endpointAddress *endp.Addr,
	payload didcomm.Payload) (response didcomm.Payload, nonce string) {

	switch endpointAddress.PlRcvr {
	case PingHandlerEndpoint:
		return mesg.PayloadCreator.New(didcomm.PayloadInit{
			ID:   payload.ID(),
			Type: payload.Type(),
			MsgInit: didcomm.MsgInit{
				Encrypted: utils.Settings.HostAddr(),
				Name:      utils.Settings.VersionInfo(),
			},
		}), nonce

	case HandlerEndpoint:
		if payload.Type() == pltype.ConnectionHandshake {
			email := payload.Message().Endpoint().Endp

			errorMsg := mesg.PayloadCreator.New(didcomm.PayloadInit{
				ID:   payload.ID(),
				Type: pltype.ConnectionHandshake,
				MsgInit: didcomm.MsgInit{
					Error: AlreadyExistError,
				},
			})

			// This IMPORTANT CHECK POINT for security check and redundancy
			if !payload.Message().ChecksumOK() ||
				!enclave.WalletKeyNotExists(email) {

				return errorMsg, nonce
			}

			agentHandler, err := anchorAgent(email)
			if err != nil {
				return errorMsg, nonce
			}
			// agentHandler will register to handlers back to us with this call
			response, _ = agentHandler.InOutPL(endpointAddress, payload)
			// we dont want to forward previous nonce to our caller in this
			// phase of the handshake procedure
			return response, nonce
		}
	}
	return mesg.PayloadCreator.New(didcomm.PayloadInit{
		ID:   payload.ID(),
		Type: payload.Type(),
		MsgInit: didcomm.MsgInit{
			Error: UnknownTypeError,
		},
	}), nonce
}

// SetSteward sets the steward agent of this Agency.
func SetSteward(st *cloud.Agent) {
	steward = st
}

// LoadRegistered is usually called only once, in the startup of the service.
func LoadRegistered(filename string) (err error) {
	utils.Settings.SetRegisterName(filename)
	err = agency.Register.Load(filename)
	if err != nil {
		return fmt.Errorf("load register: %s", err)
	}
	go func() {
		defer err2.CatchAll(func(err error) {
			glog.Fatal(err)
		}, func(exception interface{}) {
			glog.Fatal(exception)
		})

		registeredWallets := make(map[string]bool) // no duplicates

		agency.Register.EnumValues(func(rootDid string, values []string) (next bool) {
			// dont let crash on panics
			defer err2.Catch(func(err error) {
				glog.Error("agency load: ", err)
			})
			next = true // default is to continue even on error
			email := values[0]
			caDid := values[1]

			rippedEmail := strings.Replace(email, "@", "_", -1)
			walletExist := registeredWallets[rippedEmail]
			if !walletExist {
				key, err := enclave.WalletKeyByEmail(email)
				keyByDid, error2 := enclave.WalletKeyByDID(rootDid)
				if err != nil || error2 != nil {
					glog.Warning("cannot get wallet key:", err, email)
					return true
				}
				if key != keyByDid {
					glog.Warning("keys don't match", key, keyByDid)
				}

				aw := ssi.NewRawWalletCfg(rippedEmail, key)
				if !aw.Exists(false) {
					glog.Warningf("wallet %s not exist", rippedEmail)
					return true
				}

				registeredWallets[rippedEmail] = true

				agency.AddSeedHandler(caDid, cloud.NewSeedAgent(rootDid,
					caDid, aw))
			} else {
				glog.Fatal("Duplicate registered wallet!")
			}
			return true
		})
		glog.V(1).Info("LoadRegistered done")
	}()
	glog.V(1).Info("LoadRegistered kicked to start")
	return nil
}

// SetStewardFromWallet sets steward DID for us from pre-created wallet and
// named DID string.
func SetStewardFromWallet(wallet *ssi.Wallet, DID string) {
	agent := cloud.Agent{}
	agent.OpenWallet(*wallet)
	agent.SetRootDid(agent.OpenDID(DID))
	SetSteward(&agent)
}

func RegisterGobs() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}
