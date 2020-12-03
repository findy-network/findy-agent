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
	grpcAnswerTimeout = 1 * time.Hour

	// pingSATimeout is different because it is not a actual question i.e. we
	// don't need to get an answer. If we don't it just means that SA is not
	// listening and an admin can do something about it. The current might
	// actually be too long.
	pingSATimeout = 3 * time.Second
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
	if glog.V(1) {
		glog.Info("grpc SA API call:", plType, im.Info())
		glog.Infof("thread.ID: %v thread.PID: %v, pw.id: %v",
			im.Nonce(), im.SubLevelID(), im.Name())
	}

	// we don't need clone but reuse the incoming message. In future when we
	// only have gRPC based API and we don't need old indy-based messages, this
	// isn't any problem.
	om = im

	om.SetReady(false) // make sure about default behavior

	switch plType {
	case pltype.CANotifyStatus:
		// todo: we should get these in any case?
	case pltype.SAPing:
		handlePing(WDID, plType, im, om)
	case pltype.SAIssueCredentialAcceptPropose:
		handleAcceptIssuePropose(WDID, plType, im, om)
	case pltype.SAPresentProofAcceptPropose:
		handleAcceptProof(WDID, plType, im, om)
	case pltype.SAPresentProofAcceptValues:
		handleAcceptProofValues(WDID, plType, im, om)
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
			ProtocolFamily:   pltype.ProtocolTrustPing,
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

	case <-time.After(pingSATimeout):
		glog.V(1).Infof("!!!!! no answer in time (%v) for: %v",
			pingSATimeout, qid)
		om.SetReady(false)
		om.SetInfo("timeout")

	}
}

func handleAcceptIssuePropose(WDID string, plType string, im didcomm.Msg, om didcomm.Msg) {
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
			ProtocolFamily:   pltype.ProtocolIssueCredential,
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
			ProtocolFamily:   pltype.ProtocolPresentProof,
			NotificationType: plType,
			ConnectionID:     im.Name(),
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

func handleAcceptProofValues(WDID string, plType string, im didcomm.Msg, om didcomm.Msg) {
	var proof *bus.ProofVerify
	if im.ProofValues() != nil {
		proof = &bus.ProofVerify{Attrs: *im.ProofValues()}
	}
	qid := utils.UUID() // every question ID must have unique ID!
	ac := bus.WantAllAgentAnswers.AgentSendQuestion(bus.AgentQuestion{
		AgentNotify: bus.AgentNotify{ // todo: initiator is missing
			AgentKeyType: bus.AgentKeyType{
				AgentDID: WDID,
			},
			ID:               qid,
			PID:              im.SubLevelID(),
			ProtocolFamily:   pltype.ProtocolPresentProof,
			ProofVerify:      proof,
			NotificationType: plType,
			ConnectionID:     im.Name(),
			ProtocolID:       im.Nonce(),
			//Initiator:
		},
	})
	select {
	case a := <-ac:
		glog.V(1).Infof("======= got answer (%v) for: %v", a.ACK, qid)
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
