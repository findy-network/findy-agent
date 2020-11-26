package sa

import (
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/golang/glog"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
)

const (
	GRPCSA = "grpc"
)

func init() {
	Add(GRPCSA, grpcHandler)
}

// grpcHandler implements SA as a gRPC router. The blocking mechanism it uses is
// based on Go channels and to be able to "reserve" current gorountine for the
// blocking answer wait without thinking that goroutines are limited resource.
//
// During the implementation one option was to make the agency be the calling
// side, but it would make e.g. authentication an issue. By the current
// implementation we can have blocking call to make state management in the PSM
// easier and still not use too much resources. However, this is not the perfect
// solution. In the future we can make this fully async like user actions to
// mobile EAs i.e. implement SA questions as an own states in the PSM.
func grpcHandler(WDID, plType string, im didcomm.Msg) (om didcomm.Msg, err error) {
	glog.V(1).Info("grpc SA API call:", plType, im.Info())

	switch plType {
	case pltype.CANotifyStatus:

	case pltype.SAPing:
		om = im // this if from legacy impl. will check in future if needed.

		qid := utils.UUID() // every question ID must have unique ID!
		ac := bus.WantAllAgentAnswers.AgentSendQuestion(bus.AgentQuestion{
			AgentNotify: bus.AgentNotify{
				AgentKeyType: bus.AgentKeyType{
					AgentDID: WDID,
				},
				ID:               qid,
				NotificationType: plType,
				ConnectionID:     im.Thread().ID,
				ProtocolID:       im.Nonce(),
			},
		})
		// todo: add timeout & end preferable we should have a
		//  signaling mechanism if we cannot reach SA,
		//  signaling could be just webhook?
		a := <-ac

		glog.V(1).Infoln("got answer for:", qid)
		om.SetReady(a.ACK)
		om.SetInfo(a.Info)

	case pltype.SAIssueCredentialAcceptPropose:
		// todo: add real call to SA
		om = im
		// in real case, make sure data matches the credential proposal
		om.SetReady(true)

	case pltype.SAPresentProofAcceptPropose:
		// todo: add real call to SA

		om = im
		// todo: this should be get somewhere?
		attrInfo := anoncreds.AttrInfo{
			Name: "email",
		}
		reqAttrs := map[string]anoncreds.AttrInfo{
			"attr1_referent": attrInfo,
		}
		nonce := utils.NewNonceStr()
		proofRequest := anoncreds.ProofRequest{
			Name:                "FirstProofReq",
			Version:             "0.1",
			Nonce:               nonce,
			RequestedAttributes: reqAttrs,
			RequestedPredicates: map[string]anoncreds.PredicateInfo{},
		}
		reqStr := dto.ToJSON(proofRequest)
		om.SetSubMsg(mesg.SubFromJSON(reqStr))
		om.SetReady(true)
	case pltype.SAPresentProofAcceptValues:
		om = im

		// Sample how SA value verification is written
		proofJSON := dto.ToJSON(im.SubMsg())
		var proof anoncreds.Proof
		dto.FromJSONStr(proofJSON, &proof)
		emailToVerify := proof.RequestedProof.RevealedAttrs["attr1_referent"].Raw
		glog.V(1).Info("Testing mock cannot REALLY verify this: ", emailToVerify)

		om.SetReady(true)
	}
	return om, nil
}
