package sa

import (
	"time"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/golang/glog"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
)

const (
	GRPCSA = "grpc"

	// todo: this must change at the production time or put to settings/cfg
	grpcAnswerTimeout = 10 * time.Second
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
		// todo: we should get these in any case?
	case pltype.SAPing:
		// todo: what this does in really?
		om = im // this if from legacy impl. will check in future if needed.

		handlePing(WDID, plType, im, om)
	case pltype.SAIssueCredentialAcceptPropose:
		om = im
		handleIssuePropose(WDID, plType, im, om)
	case pltype.SAPresentProofAcceptPropose:
		om = im
		handleAcceptProof(WDID, plType, im, om)
	case pltype.SAPresentProofAcceptValues:
		om = im
		handleProofValues(WDID, plType, im, om)
	}
	return om, nil
}

func handlePing(WDID string, plType string, im didcomm.Msg, om didcomm.Msg) {
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
	select {
	case a := <-ac:
		glog.V(1).Infoln("got answer for:", qid)
		om.SetReady(a.ACK)
		om.SetInfo(a.Info)

	case <-time.After(grpcAnswerTimeout):
		glog.V(1).Infof("!!!!! no answer in time (%v) for: %v",
			grpcAnswerTimeout, qid)
		om.SetReady(false)
		om.SetInfo("timeout")

	}
}

func handleIssuePropose(WDID string, plType string, im didcomm.Msg, om didcomm.Msg) {
	PID := ""
	if im.SubMsg() != nil {
		PID, _ = im.SubMsg()["id"].(string)
	}
	issue := &bus.IssuePropose{
		CredDefID:  im.SubLevelID(),
		ValuesJSON: im.Info(),
	}
	qid := utils.UUID() // every question ID must have unique ID!
	ac := bus.WantAllAgentAnswers.AgentSendQuestion(bus.AgentQuestion{
		AgentNotify: bus.AgentNotify{
			AgentKeyType: bus.AgentKeyType{
				AgentDID: WDID,
			},
			ID:               qid,
			PID:              PID,
			IssuePropose:     issue,
			NotificationType: plType,
			ConnectionID:     im.Thread().ID,
			ProtocolID:       im.Nonce(),
		},
	})
	select {
	case a := <-ac:
		glog.V(1).Infoln("got answer for:", qid)
		if a.Info != im.Info() {
			om.SetInfo(a.Info)
		}
		om.SetReady(a.ACK)
	case <-time.After(grpcAnswerTimeout):
		glog.V(1).Infof("!!!!! no answer in time (%v) for: %v",
			grpcAnswerTimeout, qid)
		om.SetReady(false) // we could play with these in tests?
		om.SetInfo(im.Info())
	}
}

func handleAcceptProof(WDID string, plType string, im didcomm.Msg, om didcomm.Msg) {
	qid := utils.UUID() // every question ID must have unique ID!
	ac := bus.WantAllAgentAnswers.AgentSendQuestion(bus.AgentQuestion{
		AgentNotify: bus.AgentNotify{
			AgentKeyType: bus.AgentKeyType{
				AgentDID: WDID,
			},
			ID:               qid,
			PID:              im.SubLevelID(),
			NotificationType: plType,
			ConnectionID:     im.Thread().ID,
			ProtocolID:       im.Nonce(),
		},
	})
	select {
	case a := <-ac:
		glog.V(1).Infoln("got answer for:", qid)
		om.SetReady(a.ACK)
	case <-time.After(grpcAnswerTimeout):
		glog.V(1).Infof("!!!!! no answer in time (%v) for: %v",
			grpcAnswerTimeout, qid)
		om.SetReady(false) // we could play with these in tests?
	}
}

func handleProofValues(WDID string, plType string, im didcomm.Msg, om didcomm.Msg) {
	om = im
	PID := ""
	if im.SubMsg() != nil {
		PID, _ = im.SubMsg()["id"].(string)
	}
	var proof *bus.ProofVerify
	if im.ProofValues() != nil {
		proof = &bus.ProofVerify{Attrs: *im.ProofValues()}
	}
	qid := utils.UUID() // every question ID must have unique ID!
	ac := bus.WantAllAgentAnswers.AgentSendQuestion(bus.AgentQuestion{
		AgentNotify: bus.AgentNotify{
			AgentKeyType: bus.AgentKeyType{
				AgentDID: WDID,
			},
			ID:               qid,
			PID:              PID,
			ProofVerify:      proof,
			NotificationType: plType,
			ConnectionID:     im.Thread().ID,
			ProtocolID:       im.Nonce(),
		},
	})
	select {
	case a := <-ac:
		glog.V(1).Infoln("got answer for:", qid)
		if a.Info != im.Info() {
			om.SetInfo(a.Info)
		}
		om.SetReady(a.ACK)
	case <-time.After(grpcAnswerTimeout):
		glog.V(1).Infof("!!!!! no answer in time (%v) for: %v",
			grpcAnswerTimeout, qid)
		om.SetReady(false) // we could play with these in tests?
	}
}
