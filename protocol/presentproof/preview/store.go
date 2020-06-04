// Package preview includes helpers for Aries present proof protocol processor.
package preview

import (
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/psm"
	"github.com/optechlab/findy-go/anoncreds"
	"github.com/optechlab/findy-go/dto"
)

func StoreProofData(requestData []byte, rep *psm.PresentProofRep) {
	var proofReq anoncreds.ProofRequest
	dto.FromJSON(requestData, &proofReq)
	rep.Attributes = make([]didcomm.ProofAttribute, 0)
	for id, attr := range proofReq.RequestedAttributes {
		credDefID := ""
		if len(attr.Restrictions) > 0 {
			credDefID = attr.Restrictions[0].CredDefID
		}
		rep.Attributes = append(
			rep.Attributes,
			didcomm.ProofAttribute{ID: id, Name: attr.Name, CredDefID: credDefID},
		)
	}
}
