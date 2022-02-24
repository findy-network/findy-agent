// Package preview includes helpers for Aries present proof protocol processor.
package preview

import (
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/protocol/presentproof/data"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
)

func StoreProofData(requestData []byte, rep *data.PresentProofRep) {
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
