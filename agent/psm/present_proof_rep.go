package psm

import (
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
)

type PresentProofRep struct {
	Key        StateKey
	ProofReq   string
	Proof      string
	Values     string // currently only used for Task API to get data
	WeProposed bool
	Attributes []didcomm.ProofAttribute
}

func NewPresentProofRep(d []byte) *PresentProofRep {
	p := &PresentProofRep{}
	dto.FromGOB(d, p)
	return p
}

func (rep *PresentProofRep) Data() []byte {
	return dto.ToGOB(rep)
}

func (rep *PresentProofRep) KData() []byte {
	return rep.Key.Data()
}

const fetchMax = 2

// CreateProof is PROVER side helper.
func (rep *PresentProofRep) CreateProof(packet comm.Packet, rootDID string) (err error) {
	defer err2.Annotate("create proof", &err)

	if glog.V(6) {
		glog.Info(rep.Key, rootDID)
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

	schemasJSON := err2.String.Try(schemas(rootDID, foundSchemas))
	credDefsJSON := err2.String.Try(credDefs(rootDID, foundCredDefs))

	masterSec := err2.String.Try(packet.Receiver.MasterSecret())
	r := <-anoncreds.ProverCreateProof(w2, rep.ProofReq, reqCredJSON,
		masterSec, schemasJSON, credDefsJSON, "{}")
	err2.Check(r.Err())
	rep.Proof = r.Str1()
	return nil
}

func (rep *PresentProofRep) processAttributes(w2 int,
	proofReq anoncreds.ProofRequest) (anoncreds.RequestedCredentials, []anoncreds.Credentials) {

	r := <-anoncreds.ProverSearchCredentialsForProofReq(w2, rep.ProofReq, findy.NullString)
	err2.Check(r.Err())
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
			err2.Check(r.Err())
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
			} else {
				break
			}
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
			err2.Check(r.Err())
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
			} else {
				break
			}
		}
	}

	r = <-anoncreds.ProverCloseCredentialsSearchForProofReq(searchHandle)
	err2.Check(r.Err())
	return reqCred, allCredInfos
}

func credDefs(DID string, credDefIDs map[string]struct{}) (cJSON string, err error) {
	defer err2.Annotate("cred defs", &err)

	credDefs := make(map[string]map[string]interface{}, len(credDefIDs))
	for cdID := range credDefIDs {
		credDef := err2.String.Try(ssi.CredDefFromLedger(DID, cdID))
		credDefObject := map[string]interface{}{}
		dto.FromJSONStr(credDef, &credDefObject)
		credDefs[cdID] = credDefObject
	}
	credDefsJSON := dto.ToJSON(credDefs)
	return credDefsJSON, nil
}

func schemas(DID string, schemaIDs map[string]struct{}) (sJSON string, err error) {
	defer err2.Annotate("get schemas", &err)

	schemas := make(map[string]map[string]interface{}, len(schemaIDs))
	for schemaID := range schemaIDs {
		sch := ssi.Schema{ID: schemaID}
		err2.Check(sch.FromLedger(DID))
		schemaObject := map[string]interface{}{}
		dto.FromJSONStr(sch.LazySchema(), &schemaObject)
		schemas[schemaID] = schemaObject
	}
	schemasJSON := dto.ToJSON(schemas)
	return schemasJSON, nil
}

func (rep *PresentProofRep) VerifyProof(packet comm.Packet) (ack bool, err error) {
	defer err2.Annotate("verify proof", &err)

	var proof anoncreds.Proof
	dto.FromJSONStr(rep.Proof, &proof)

	rootDID := packet.Receiver.RootDid().Did()
	schemaIDs := getSchemaIDs(proof.Identifiers)
	schemasJSON := err2.String.Try(schemas(rootDID, schemaIDs))

	credDefIDs := getCredDefIDs(proof.Identifiers)
	credDefsJSON := err2.String.Try(credDefs(rootDID, credDefIDs))

	r := <-anoncreds.VerifierVerifyProof(rep.ProofReq, rep.Proof, schemasJSON, credDefsJSON, "{}", "{}")
	err2.Check(r.Err())
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
