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

	"github.com/findy-network/findy-agent/agent/accessmgr"
	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/findy-network/findy-agent/method"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

var Hub *Agency

func init() {
	Hub = new(Agency)
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

// AnchorAgent Builds new trust anchor agent and its wallet. This is quite slow process. In
// future we could build them in advance to pool where we could allocate them
// when needed. Needs to wallet renaming or indexing.
func AnchorAgent(agentName, seed string) (agent *cloud.Agent, caDid core.DID, err error) {
	defer err2.Handle(&err, "create anchor")

	key := try.To1(enclave.NewWalletKey(agentName))
	defer func() {
		key = ""
	}()

	// Build new agent with wallet
	agent = new(cloud.Agent)
	ca := agent
	aw := ssi.NewRawWalletCfg(agentName, key)
	walletAlreadyExists := aw.Create()
	assert.P.Truef(!walletAlreadyExists,
		"wallet (%s) cannot exist when onboarding", agentName)
	agent.OpenWallet(*aw)

	glog.V(10).Infof("--- Using seed '%s' in anchor agent", seed)
	anchorDid := try.To1(agent.NewDID(method.TypeSov, seed))
	if steward != nil {
		assert.P.True(seed == "", "seed should be empty when agency is operating with steward")

		indyAnchor := anchorDid.(*ssi.DID)
		// Promote new agent by Trusted Anchor DID if steward is available
		try.To(steward.SendNYM(indyAnchor, steward.RootDid().Did(),
			findy.NullString, "TRUST_ANCHOR"))
	}

	// Use the anchor DID as a submitter/root DID to Ledger
	agent.SetRootDid(anchorDid)

	// Add newly created agent's managed wallet to access mgr's backup list.
	// The backup is taken only once because rest of the agent's wallet data is
	// in the worker wallet. By this we don't keep taking unnecessary backups
	// from a pairwise only CA wallet in continuous backup process.
	accessmgr.Send(agent.WalletH)

	glog.V(2).Infof("--- Saving enclave key for %s", anchorDid.KID())
	caDid = try.To1(agent.NewDID(method.TypeSov, ""))
	try.To(enclave.SetKeysDID(key, caDid.Did()))

	glog.V(2).Infof("Creating a master secret into agent wallet (%s)", caDid.Did())
	masterSec, err := enclave.NewWalletMasterSecret(caDid.Did())
	assert.NoError(err)
	r := <-anoncreds.ProverCreateMasterSecret(ca.Wallet(), masterSec)
	assert.NoError(r.Err())
	assert.Equal(masterSec, r.Str1())

	return agent, caDid, nil
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
		defer err2.Catch(func(err error) {
			glog.Fatal(err)
		}, func(exception interface{}) {
			glog.Fatal(exception)
		})

		// for book keeping we don't allow duplicates and because registry is
		// still JSON file there is possibility for a human error.
		alreadyRegistered := make(map[string]bool)

		agency.Register.EnumValues(func(caDID string, values []string) (next bool) {
			email := values[0]
			rootDid := values[1]
			caVerKey := ""
			if len(values) == 3 {
				caVerKey = values[2]
			}
			name := strings.Replace(email, "@", "_", -1)

			// don't let crash on panics
			defer err2.Catch(func(err error) {
				glog.Errorf("error: %s in agency load (email %s,DID:%s)",
					err, email, caDID)
			})

			if !alreadyRegistered[name] {
				glog.V(2).Infof("Loading enclave key for root: %s, ca: %s", rootDid, caDID)

				key := try.To1(enclave.WalletKeyByEmail(email))
				keyByDid := try.To1(enclave.WalletKeyByDID(caDID))

				if key != keyByDid {
					// key values are left out from logs in purpose
					glog.Warningf("-------------------------------\n"+
						"key by email (%s) don't match key by caDid\n"+
						"using key by email", email)
				}

				aw := ssi.NewRawWalletCfg(name, key)
				wantToSeeWorker := false
				if !aw.Exists(wantToSeeWorker) {
					glog.Warningf("wallet '%s' not exist. Skipping this"+
						" agent allocation and move to next", name)
					return true
				}

				alreadyRegistered[name] = true

				agency.AddSeedHandler(caDID,
					cloud.NewSeedAgent(rootDid, caDID, caVerKey, aw))
			} else {
				glog.Fatal("Duplicate registered wallet!")
			}
			return true
		})
		glog.V(1).Info("LoadRegistered done")
		agency.Ready.RegisteringComplete()
	}()
	glog.V(1).Info("LoadRegistered kicked to start")
	return nil
}

// SetStewardFromWallet sets steward DID for us from pre-created wallet and
// named DID string.
func SetStewardFromWallet(wallet *ssi.Wallet, DID string) (stwd *cloud.Agent) {
	agent := new(cloud.Agent)
	agent.OpenWallet(*wallet)
	agent.SetRootDid(agent.OpenDID(DID))
	SetSteward(agent)
	return agent
}

func RegisterGobs() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}
