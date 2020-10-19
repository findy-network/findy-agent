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
	GRPCSA = "grpc_sa"
)

func init() {
	Add(GRPCSA, grpcHandler)
}

// todo:
//  1. notify agent listener = send question to controller via channel (what channel?)
//  2. wait reply with the channel (what channel to listen) to arrive
//  3. channel communication would be same as listeners but other way around
//     we would register a question with UUID and reply channel to wait
//     when outside get notification about the question it can answer to it
//     by the UUID and reply would be delivered to the proper channel
//     we should decide if it's better to have question listener different
//     than the normal agent listener what we now have.
// TODO:
//  1. call directly the SA who has given us gRPC endpoint to call
// how to authenticate? we should make it server, but if we don't make this isn't so easy
//

func grpcHandler(WDID, plType string, im didcomm.Msg) (om didcomm.Msg, err error) {
	glog.V(1).Info("grpc SA API call:", plType, im.Info())

	switch plType {
	case pltype.CANotifyStatus:

	case pltype.SAPing:
		om = im
		qid := utils.UUID()
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
		a := <-ac
		glog.V(1).Infoln("got answer for:", qid)
		om.SetReady(a.ACK)
		om.SetInfo(a.Info)

	case pltype.SAIssueCredentialAcceptPropose:
		om = im
		// in real case, make sure data matches the credential proposal
		om.SetReady(true)

	case pltype.SAPresentProofAcceptPropose:
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
