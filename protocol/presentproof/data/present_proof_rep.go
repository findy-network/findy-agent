package data

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/vc"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

const bucketType = psm.BucketPresentProof

type PresentProofRep struct {
	psm.StateKey
	ProofReq   string
	Proof      string
	Values     []string // TODO: reserved for indy-WQL
	WeProposed bool
	Attributes []didcomm.ProofAttribute
}

func init() {
	psm.Creator.Add(bucketType, NewPresentProofRep)
}

func NewPresentProofRep(d []byte) psm.Rep {
	p := &PresentProofRep{}
	dto.FromGOB(d, p)
	return p
}

func (rep *PresentProofRep) Key() psm.StateKey {
	return rep.StateKey
}

func (rep *PresentProofRep) Data() []byte {
	return dto.ToGOB(rep)
}

func (rep *PresentProofRep) Type() byte {
	return bucketType
}

const fetchMax = 2

// CreateProof is PROVER side helper.
func (rep *PresentProofRep) CreateProof(packet comm.Packet, rootDID string) (err error) {
	defer err2.Handle(&err, "create proof")

	if glog.V(6) {
		glog.Info(rep.Key(), rootDID)
		glog.Info("+++ proof req:\n", rep.ProofReq)
	}

	w2 := packet.Receiver.Wallet()
	var proofReq anoncreds.ProofRequest
	dto.FromJSONStr(rep.ProofReq, &proofReq)

	reqCred, allCredInfos := rep.processAttributes(w2, proofReq)
	reqCredJSON := dto.ToJSON(reqCred)

	// get schemas and cred defs for all requested attributes from the ledger.
	foundSchemas := make(map[string]struct{}, len(allCredInfos))
	foundCredDefs := make(map[string]struct{}, len(allCredInfos))
	for _, v := range allCredInfos {
		foundSchemas[v.CredInfo.SchemaID] = struct{}{}
		foundCredDefs[v.CredInfo.CredDefID] = struct{}{}
	}

	schemasJSON := try.To1(schemas(rootDID, foundSchemas))
	credDefsJSON := try.To1(credDefs(rootDID, foundCredDefs))

	masterSec := try.To1(packet.Receiver.MasterSecret())
	r := <-anoncreds.ProverCreateProof(w2, rep.ProofReq, reqCredJSON,
		masterSec, schemasJSON, credDefsJSON, "{}")
	try.To(r.Err())
	rep.Proof = r.Str1()
	return nil
}

func (rep *PresentProofRep) processAttributes(
	w2 int,
	proofReq anoncreds.ProofRequest,
) (
	anoncreds.RequestedCredentials,
	[]anoncreds.Credentials,
) {
	// TODO: build from rep.Values, see Go findy-wrapper-go, anoncreds_test.go
	wql := findy.NullString
	r := <-anoncreds.ProverSearchCredentialsForProofReq(w2, rep.ProofReq, wql)
	try.To(r.Err())
	searchHandle := r.Handle()

	reqCred := anoncreds.RequestedCredentials{
		SelfAttestedAttributes: make(map[string]string),
		RequestedAttributes:    make(map[string]anoncreds.RequestedAttrObject),
		RequestedPredicates:    make(map[string]anoncreds.RequestedPredObject),
	}

	allCredInfos := make([]anoncreds.Credentials, 0, fetchMax)

	// gather cred infos for requested attributes.
	for attrRef, aInfo := range proofReq.RequestedAttributes {
		foundAlready := false
		for {
			r = <-anoncreds.ProverFetchCredentialsForProofReq(searchHandle,
				attrRef, fetchMax)
			try.To(r.Err())
			credentials := r.Str1()
			credInfo := make([]anoncreds.Credentials, 0, fetchMax)
			dto.FromJSONStr(credentials, &credInfo)
			allCredInfos = append(allCredInfos, credInfo...)

			// we add the first found cred info of the patch to requested
			// attribute we are processing
			if len(credInfo) > 0 {
				foundAlready = true
				obj := anoncreds.RequestedAttrObject{
					CredID:    credInfo[0].CredInfo.Referent,
					Revealed:  true,
					Timestamp: nil,
				}
				reqCred.RequestedAttributes[attrRef] = obj
			}

			if len(credInfo) == fetchMax {
				glog.V(1).Info("--- There's more cred infos for attr ---")
				continue
			}
			break
		}
		selfAttestedNeedsToBeSet := !foundAlready && len(aInfo.Restrictions) == 0

		if selfAttestedNeedsToBeSet {
			glog.V(1).Info("Self attested attr:", aInfo.Name)
			reqCred.SelfAttestedAttributes[attrRef] = "my self-attested value"
		}
	}

	// gather cred infos for predicated attributes
	for predicateRef := range proofReq.RequestedPredicates {
		for {
			r = <-anoncreds.ProverFetchCredentialsForProofReq(searchHandle,
				predicateRef, fetchMax)
			try.To(r.Err())
			credentials := r.Str1()
			credInfo := make([]anoncreds.Credentials, 0, fetchMax)
			dto.FromJSONStr(credentials, &credInfo)
			allCredInfos = append(allCredInfos, credInfo...)

			// we add the first found cred info of the patch to requested
			// attribute we are processing
			if len(credInfo) > 0 {
				obj := anoncreds.RequestedPredObject{
					CredID:    credInfo[0].CredInfo.Referent,
					Timestamp: nil,
				}
				reqCred.RequestedPredicates[predicateRef] = obj
			}

			if len(credInfo) == fetchMax {
				glog.V(1).Info("--- There's more pred infos for attr ---")
				continue
			}
			break
		}
	}

	r = <-anoncreds.ProverCloseCredentialsSearchForProofReq(searchHandle)
	try.To(r.Err())
	return reqCred, allCredInfos
}

func credDefs(DID string, credDefIDs map[string]struct{}) (cJSON string, err error) {
	defer err2.Handle(&err, "cred defs")

	credDefs := make(map[string]map[string]interface{}, len(credDefIDs))
	for cdID := range credDefIDs {
		credDef := try.To1(vc.CredDefFromLedger(DID, cdID))
		credDefObject := map[string]interface{}{}
		dto.FromJSONStr(credDef, &credDefObject)
		credDefs[cdID] = credDefObject
	}
	credDefsJSON := dto.ToJSON(credDefs)
	return credDefsJSON, nil
}

func schemas(DID string, schemaIDs map[string]struct{}) (sJSON string, err error) {
	defer err2.Handle(&err, "get schemas")

	schemas := make(map[string]map[string]interface{}, len(schemaIDs))
	for schemaID := range schemaIDs {
		sch := vc.Schema{ID: schemaID}
		try.To(sch.FromLedger(DID))
		schemaObject := map[string]interface{}{}
		dto.FromJSONStr(sch.LazySchema(), &schemaObject)
		schemas[schemaID] = schemaObject
	}
	schemasJSON := dto.ToJSON(schemas)
	return schemasJSON, nil
}

func (rep *PresentProofRep) VerifyProof(packet comm.Packet) (ack bool, err error) {
	defer err2.Handle(&err, "verify proof")

	var proof anoncreds.Proof
	dto.FromJSONStr(rep.Proof, &proof)

	rootDID := packet.Receiver.RootDid().Did()
	schemaIDs := getSchemaIDs(proof.Identifiers)
	schemasJSON := try.To1(schemas(rootDID, schemaIDs))

	credDefIDs := getCredDefIDs(proof.Identifiers)
	credDefsJSON := try.To1(credDefs(rootDID, credDefIDs))

	r := <-anoncreds.VerifierVerifyProof(rep.ProofReq, rep.Proof, schemasJSON, credDefsJSON, "{}", "{}")
	try.To(r.Err())
	return r.Yes(), nil
}

func getSchemaIDs(identifiers []anoncreds.IdentifiersObj) map[string]struct{} {
	IDs := make(map[string]struct{}, len(identifiers))
	for _, v := range identifiers {
		IDs[v.SchemaID] = struct{}{}
	}
	return IDs
}

func getCredDefIDs(identifiers []anoncreds.IdentifiersObj) map[string]struct{} {
	IDs := make(map[string]struct{}, len(identifiers))
	for _, v := range identifiers {
		IDs[v.CredDefID] = struct{}{}
	}
	return IDs
}

func GetPresentProofRep(key psm.StateKey) (rep *PresentProofRep, err error) {
	defer err2.Handle(&err)

	res := try.To1(psm.GetRep(bucketType, key))

	// Allow not found
	if res == nil {
		return
	}

	var ok bool
	rep, ok = res.(*PresentProofRep)

	assert.That(ok, "present proof type mismatch")

	return rep, nil
}
