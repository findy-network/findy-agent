// Package verifier includes Aries protocol handlers for a verifier.
package verifier

import (
	"strconv"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/protocol/presentproof/data"
	"github.com/findy-network/findy-agent/protocol/presentproof/preview"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/presentproof"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

const ackOK = "OK"

func generateProofRequest(proofTask *presentproof.Propose) *anoncreds.ProofRequest {
	reqAttrs := make(map[string]anoncreds.AttrInfo)
	for index, attr := range proofTask.PresentationProposal.Attributes {
		restrictions := make([]anoncreds.Filter, 0)
		if attr.CredDefID != "" {
			restrictions = append(restrictions, anoncreds.Filter{CredDefID: attr.CredDefID})
		}
		id := "attr_referent_" + strconv.Itoa(index+1)
		reqAttrs[id] = anoncreds.AttrInfo{
			Name:         attr.Name,
			Restrictions: restrictions,
		}
	}
	reqPredicates := make(map[string]anoncreds.PredicateInfo)
	if proofTask.PresentationProposal.Predicates != nil {
		for index, predicate := range proofTask.PresentationProposal.Predicates {
			// TODO: restrictions
			id := "predicate_" + strconv.Itoa(index+1)
			value, _ := strconv.ParseInt(predicate.Threshold, 10, 64) // TODO
			reqPredicates[id] = anoncreds.PredicateInfo{
				Name:   predicate.Name,
				PType:  predicate.Predicate,
				PValue: int(value),
			}
		}
	}
	return &anoncreds.ProofRequest{
		Name:                "ProofReq",
		Version:             "0.1",
		Nonce:               utils.NewNonceStr(),
		RequestedAttributes: reqAttrs,
		RequestedPredicates: reqPredicates,
	}
}

// HandleProposePresentation is a protocol handler function at VERIFIER side.
func HandleProposePresentation(packet comm.Packet) (err error) {
	var sendNext, waitingNext string
	if packet.Receiver.AutoPermission() {
		sendNext = pltype.PresentProofRequest
		waitingNext = pltype.PresentProofPresentation
	} else {
		sendNext = pltype.Nothing
		waitingNext = pltype.PresentProofUserAction
	}

	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    sendNext,
		WaitingNext: waitingNext,
		SendOnNACK:  pltype.PresentProofNACK,
		TaskHeader:  &comm.TaskHeader{UserActionPLType: pltype.SAPresentProofAcceptPropose},
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof propose handler", &err)

			agent := packet.Receiver
			meDID := agent.Trans().MessagePipe().In.Did()
			key := psm.StateKey{DID: meDID, Nonce: im.Thread().ID}

			propose := im.FieldObj().(*presentproof.Propose)
			proofReq := generateProofRequest(propose)
			reqStr := dto.ToJSON(proofReq)

			attributes := make([]didcomm.ProofAttribute, 0)
			for _, attr := range propose.PresentationProposal.Attributes {
				attributes = append(attributes, didcomm.ProofAttribute{
					Name:      attr.Name,
					CredDefID: attr.CredDefID,
				})
			}

			rep := &data.PresentProofRep{
				StateKey:   key,
				ProofReq:   reqStr,
				Attributes: attributes,
			}
			err2.Check(psm.AddRep(rep))

			req, autoAccept := om.FieldObj().(*presentproof.Request)
			if autoAccept {
				req.RequestPresentations = presentproof.NewRequestPresentation(
					pltype.LibindyRequestPresentationID, []byte(reqStr))
			}

			return true, nil
		},
	})
}

func ContinueProposePresentation(ca comm.Receiver, im didcomm.Msg) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	err2.Check(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.PresentProofRequest,
		WaitingNext: pltype.PresentProofPresentation,
		SendOnNACK:  pltype.PresentProofNACK,
		Transfer: func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof propose user action handler", &err)

			// Does user allow continue?
			iMsg := im.(didcomm.Msg)
			if !iMsg.Ready() {
				glog.Warning("user doesn't accept proof propose")
				return false, nil
			}

			// TODO: support changing proof req

			repK := psm.NewStateKey(ca, im.Thread().ID)
			rep := e2.PresentProofRep.Try(data.GetPresentProofRep(repK))

			req := om.FieldObj().(*presentproof.Request) // query interface
			req.RequestPresentations = presentproof.NewRequestPresentation(
				pltype.LibindyRequestPresentationID, []byte(rep.ProofReq))

			return true, nil
		},
	}))
}

// HandlePresentation is a protocol handler function at VERIFIER side for handling
// proof presentation.
func HandlePresentation(packet comm.Packet) (err error) {
	var sendNext, waitingNext string
	if packet.Receiver.AutoPermission() {
		sendNext = pltype.PresentProofACK
		waitingNext = pltype.Terminate
	} else {
		sendNext = pltype.Nothing
		waitingNext = pltype.PresentProofUserAction
	}

	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    sendNext,
		WaitingNext: waitingNext,
		SendOnNACK:  pltype.PresentProofNACK,
		TaskHeader:  &comm.TaskHeader{UserActionPLType: pltype.SAPresentProofAcceptValues},
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof presentation handler", &err)

			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)
			rep := e2.PresentProofRep.Try(data.GetPresentProofRep(repK))

			// 1st, verify the proof by our selves
			pres := im.FieldObj().(*presentproof.Presentation)
			data := err2.Bytes.Try(presentproof.Proof(pres))
			rep.Proof = string(data)

			if !err2.Bool.Try(rep.VerifyProof(packet)) {
				glog.Errorf("Cannot verify proof (nonce:%v) terminating presentation protocol", im.Thread().ID)
				return false, nil
			}

			preview.StoreProofData([]byte(rep.ProofReq), rep)

			var proof anoncreds.Proof
			dto.FromJSON(data, &proof)
			for index, attr := range rep.Attributes {
				rep.Attributes[index].Value = proof.RequestedProof.RevealedAttrs[attr.ID].Raw
			}

			err2.Check(psm.AddRep(rep))

			// Autoaccept -> all checks done, let's send ACK
			ackMsg, autoAccept := om.FieldObj().(*common.Ack)
			if autoAccept {
				ackMsg.Status = ackOK
			}

			return true, nil
		},
	})
}

func ContinueHandlePresentation(ca comm.Receiver, im didcomm.Msg) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	err2.Check(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.PresentProofACK,
		WaitingNext: pltype.Terminate,
		SendOnNACK:  pltype.PresentProofNACK,
		Transfer: func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof values user action handler", &err)

			// Does user allow continue?
			iMsg := im.(didcomm.Msg)
			if !iMsg.Ready() {
				glog.Warning("user doesn't accept proof values")
				return false, nil
			}

			// All checks done, let's send ACK
			ackMsg := om.FieldObj().(*common.Ack)
			ackMsg.Status = ackOK

			return true, nil
		},
	}))
}
