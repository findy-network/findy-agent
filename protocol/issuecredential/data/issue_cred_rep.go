package data

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/ledger"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

const bucketType = psm.BucketIssueCred

type IssueCredRep struct {
	StateKey    psm.StateKey
	Timestamp   int64
	CredDefID   string
	CredDef     string
	CredOffer   string
	CredReqMeta string
	Values      string
	Attributes  []didcomm.CredentialAttribute
}

func init() {
	psm.Creator.Add(bucketType, NewIssueCredRep)
}

func NewIssueCredRep(d []byte) psm.Rep {
	p := &IssueCredRep{}
	dto.FromGOB(d, p)
	return p
}

func (rep *IssueCredRep) Key() *psm.StateKey {
	return &rep.StateKey
}

func (rep *IssueCredRep) Data() []byte {
	return dto.ToGOB(rep)
}

func (rep *IssueCredRep) Type() byte {
	return bucketType
}

// BuildCredRequest builds credential request which is PROVER/HOLDER SIDE
// action.
func (rep *IssueCredRep) BuildCredRequest(packet comm.Packet) (cr string, err error) {
	defer err2.Annotate("build cred req", &err)

	a := packet.Receiver
	w := a.Wallet()
	masterSecID := err2.String.Try(a.MasterSecret())

	// Get CRED DEF from the ledger
	_, rep.CredDef, err = ledger.ReadCredDef(a.Pool(), a.RootDid().Did(), rep.CredDefID)
	err2.Check(err)

	// build credential request to send back to an issuer
	r := <-anoncreds.ProverCreateCredentialReq(w, a.RootDid().Did(), rep.CredOffer,
		rep.CredDef, masterSecID)
	err2.Check(r.Err())
	cr = r.Str1()
	rep.CredReqMeta = r.Str2()
	return cr, nil
}

// IssuerBuildCred builds credentials in -- ISSUER SIDE --. Note! values are
// needed here!!
func (rep *IssueCredRep) IssuerBuildCred(packet comm.Packet, credReq string) (c string, err error) {
	defer err2.Annotate("build cred req", &err)

	ca := packet.Receiver.MyCA() // we need to use CA here for proper rights.
	w := ca.Wallet()

	r := <-anoncreds.IssuerCreateCredential(w, rep.CredOffer, credReq, rep.Values,
		findy.NullString, findy.NullHandle)
	err2.Check(r.Err())
	c = r.Str1()
	return c, nil
}

// StoreCred saves the credential to wallet which is prover/holder side action.
func (rep *IssueCredRep) StoreCred(packet comm.Packet, cred string) error {
	a := packet.Receiver
	w := a.Wallet()
	r := <-anoncreds.ProverStoreCredential(w, findy.NullString, rep.CredReqMeta, cred, rep.CredDef, findy.NullString)
	return r.Err()
}

func GetIssueCredRep(key *psm.StateKey) (rep *IssueCredRep, err error) {
	err2.Return(&err)

	var res psm.Rep
	res, err = psm.GetRep(bucketType, *key)
	err2.Check(err)

	// Allow not found
	if res == nil {
		return
	}

	var ok bool
	rep, ok = res.(*IssueCredRep)

	assert.D.True(ok, "issue cred type mismatch")

	return rep, nil
}
