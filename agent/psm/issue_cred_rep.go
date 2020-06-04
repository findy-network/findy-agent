package psm

import (
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-go"
	"github.com/optechlab/findy-go/anoncreds"
	"github.com/optechlab/findy-go/dto"
	"github.com/optechlab/findy-go/ledger"
)

type IssueCredRep struct {
	Key         StateKey
	Timestamp   int64
	CredDefID   string
	CredDef     string
	CredOffer   string
	CredReqMeta string
	Values      string
	Attributes  []didcomm.CredentialAttribute
}

func NewIssueCredRep(d []byte) *IssueCredRep {
	p := &IssueCredRep{}
	dto.FromGOB(d, p)
	return p
}

func (rep *IssueCredRep) Data() []byte {
	return dto.ToGOB(rep)
}

func (rep *IssueCredRep) KData() []byte {
	return rep.Key.Data()
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
