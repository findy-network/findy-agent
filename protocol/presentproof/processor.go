// Package presentproof is Aries protocol processor for present proof protocol.
package presentproof

import (
	"strconv"

	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/e2"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/prot"
	"github.com/optechlab/findy-agent/agent/psm"
	"github.com/optechlab/findy-agent/agent/utils"
	"github.com/optechlab/findy-agent/protocol/presentproof/prover"
	"github.com/optechlab/findy-agent/protocol/presentproof/verifier"
	"github.com/optechlab/findy-agent/std/presentproof"
	"github.com/optechlab/findy-go/anoncreds"
	"github.com/optechlab/findy-go/dto"
)

type statusPresentProof struct {
	Attributes []didcomm.ProofAttribute `json:"attributes"`
}

var presentProofProcessor = comm.ProtProc{
	Starter:     startProofProtocol,
	Continuator: userActionProofPresentation,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerPresentProofPropose:      verifier.HandleProposePresentation,
		pltype.HandlerPresentProofRequest:      prover.HandleRequestPresentation,
		pltype.HandlerPresentProofPresentation: verifier.HandlePresentation,
		pltype.HandlerPresentProofACK:          handleProofACK,
		pltype.HandlerPresentProofNACK:         handleProofNACK,
	},
	Status: getPresentProofStatus,
}

func init() {
	prot.AddStarter(pltype.CAProofPropose, presentProofProcessor)
	prot.AddStarter(pltype.CAProofRequest, presentProofProcessor)
	prot.AddContinuator(pltype.CAContinuePresentProofProtocol, presentProofProcessor)
	prot.AddStatusProvider(pltype.ProtocolPresentProof, presentProofProcessor)
	comm.Proc.Add(pltype.ProtocolPresentProof, presentProofProcessor)
}

func generateProofRequest(t *comm.Task) *anoncreds.ProofRequest {
	reqAttrs := make(map[string]anoncreds.AttrInfo)
	for index, attr := range *t.ProofAttrs {
		restrictions := make([]anoncreds.Filter, 0)
		if attr.CredDefID != "" {
			restrictions = append(restrictions, anoncreds.Filter{CredDefID: attr.CredDefID})
		}
		reqAttrs["attr"+strconv.Itoa(index+1)+"_referent"] = anoncreds.AttrInfo{
			Name:         attr.Name,
			Restrictions: restrictions,
		}
	}
	return &anoncreds.ProofRequest{
		Name:                "ProofReq",
		Version:             "0.1",
		Nonce:               utils.NewNonceStr(),
		RequestedAttributes: reqAttrs,
		RequestedPredicates: map[string]anoncreds.PredicateInfo{}, // TODO
	}
}

func startProofProtocol(ca comm.Receiver, t *comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	switch t.TypeID {
	case pltype.CAProofPropose: // ----- prover will start -----
		err2.Check(prot.StartPSM(prot.Initial{
			SendNext:    pltype.PresentProofPropose,
			WaitingNext: pltype.PresentProofRequest,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) error {
				// Note!! StartPSM() sends certain Task fields to other end
				// as PL.Message like msg.ID, .SubMsg, .Info

				// Check if we have previous thread id to transfer
				propose := msg.FieldObj().(*presentproof.Propose)
				if id, ok := t.Msg["id"]; ok {
					propose.Thread.PID = id.(string)
				}

				rep := &psm.PresentProofRep{
					Key:        key,
					Values:     t.Info,
					WeProposed: true,
				}
				return psm.AddPresentProofRep(rep)
			},
		}))
	case pltype.CAProofRequest: // ----- verifier will start -----
		err2.Check(prot.StartPSM(prot.Initial{
			SendNext:    pltype.PresentProofRequest,
			WaitingNext: pltype.PresentProofPresentation,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) error {
				// We are started by verifier aka SA, Proof Request comes
				// as startup argument, no need to call SA API to get it.
				// Notice that Proof Req has Nonce, we use the same one for
				// the protocol. UPDATE! After use of Aries message format,
				// we cannot share same Nonce with the proof and messages
				// here. StartPSM() sends certain Task fields to other end
				// as PL.Message
				proofRequest := generateProofRequest(t)
				// get proof req from task came in
				proofReqStr := dto.ToJSON(proofRequest)

				// set proof req to outgoing request message
				req := msg.FieldObj().(*presentproof.Request)
				req.RequestPresentations = presentproof.NewRequestPresentation(
					utils.UUID(), []byte(proofReqStr))

				// create Rep and save it for PSM to run protocol
				rep := &psm.PresentProofRep{
					Key:      key,
					Values:   t.Info,      // Verifier cannot provide this..
					ProofReq: proofReqStr, //  .. but it gives this one.
				}
				return psm.AddPresentProofRep(rep)
			},
		}))
	default:
		glog.Error("unsupported protocol start api type ")
	}
}

func userActionProofPresentation(ca comm.Receiver, im didcomm.Msg) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	err2.Check(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.PresentProofPresentation,
		WaitingNext: pltype.PresentProofACK,
		SendOnNACK:  pltype.PresentProofNACK,
		Transfer: func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof user action handler", &err)

			// Does user allow continue?
			iMsg := im.(didcomm.Msg)
			if !iMsg.Ready() {
				glog.Warning("user doesn't accept proof")
				return false, nil
			}

			// We continue, get previous data, create the proof and send it
			agent := wa
			repK := psm.NewStateKey(agent, im.Thread().ID)
			rep := e2.PresentProofRep.Try(psm.GetPresentProofRep(repK))

			err2.Check(rep.CreateProof(comm.Packet{Receiver: agent}, repK.DID))
			// save created proof to Representative
			err2.Check(psm.AddPresentProofRep(rep))

			pres := om.FieldObj().(*presentproof.Presentation)
			pres.PresentationAttaches = presentproof.NewPresentationAttach(
				pltype.LibindyPresentationID, []byte(rep.Proof))

			return true, nil
		},
	}))
}

// handleProofACK is a protocol handler func at PROVER side.
// Even the inner handler does nothing, the execution of PSM transition
// terminates the state machine which triggers the notification system to the EA
// side.
func handleProofACK(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate,
		WaitingNext: pltype.Terminate,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof ACK handler", &err)
			return true, nil
		},
	})
}

// handleProofNACK is a protocol handler func at PROVER (for this version) side.
// Even the inner handler does nothing, the execution of PSM transition
// terminates the state machine which triggers the notification system to the EA
// side.
func handleProofNACK(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate,
		WaitingNext: pltype.Terminate,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof NACK handler", &err)
			return false, nil
		},
	})
}

func getPresentProofStatus(workerDID string, taskID string) interface{} {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Failed to set present proof status: ", err)
	})
	key := &psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}

	proofRep, err := psm.GetPresentProofRep(*key)
	err2.Check(err)

	if proofRep != nil {
		return statusPresentProof{
			Attributes: proofRep.Attributes,
		}
	}

	return nil
}
