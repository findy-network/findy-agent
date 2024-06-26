// Package presentproof is Aries protocol processor for present proof protocol.
package presentproof

import (
	"encoding/gob"
	"strconv"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/protocol/presentproof/data"
	"github.com/findy-network/findy-agent/protocol/presentproof/prover"
	"github.com/findy-network/findy-agent/protocol/presentproof/verifier"
	"github.com/findy-network/findy-agent/std/presentproof"
	"github.com/findy-network/findy-common-go/dto"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type taskPresentProof struct {
	comm.TaskBase
	Comment         string
	ProofAttrs      []didcomm.ProofAttribute
	ProofPredicates []didcomm.ProofPredicate
}

type continuatorFunc func(ca comm.Receiver, im didcomm.Msg)

var presentProofProcessor = comm.ProtProc{
	Creator:     createPresentProofTask,
	Starter:     startProofProtocol,
	Continuator: continueProtocol,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerPresentProofPropose:      verifier.HandleProposePresentation,
		pltype.HandlerPresentProofRequest:      prover.HandleRequestPresentation,
		pltype.HandlerPresentProofPresentation: verifier.HandlePresentation,
		pltype.HandlerPresentProofACK:          handleProofACK,
		pltype.HandlerPresentProofNACK:         handleProofNACK,
	},
	FillStatus: fillPresentProofStatus,
}

func init() {
	gob.Register(&taskPresentProof{})
	prot.AddCreator(pltype.ProtocolPresentProof, presentProofProcessor)
	prot.AddStarter(pltype.CAProofPropose, presentProofProcessor)
	prot.AddStarter(pltype.CAProofRequest, presentProofProcessor)
	prot.AddContinuator(pltype.CAContinuePresentProofProtocol, presentProofProcessor)
	prot.AddStatusProvider(pltype.ProtocolPresentProof, presentProofProcessor)
	comm.Proc.Add(pltype.ProtocolPresentProof, presentProofProcessor)
}

func createPresentProofTask(header *comm.TaskHeader, protocol *pb.Protocol) (t comm.Task, err error) {
	defer err2.Handle(&err, "createIssueCredentialTask")

	var proofAttrs []didcomm.ProofAttribute
	var proofPredicates []didcomm.ProofPredicate
	if protocol != nil {
		proof := protocol.GetPresentProof()
		assert.That(proof != nil, "present proof data missing")
		assert.That(
			protocol.GetRole() == pb.Protocol_INITIATOR || protocol.GetRole() == pb.Protocol_ADDRESSEE,
			"role is needed for proof protocol")

		// attributes - mandatory
		if proof.GetAttributesJSON() != "" {
			dto.FromJSONStr(proof.GetAttributesJSON(), &proofAttrs)
			glog.V(3).Infoln("set proof attrs from json:", proof.GetAttributesJSON())
		} else {
			assert.That(proof.GetAttributes() != nil, "present proof attributes data missing")
			proofAttrs = make([]didcomm.ProofAttribute, len(proof.GetAttributes().GetAttributes()))
			for i, attribute := range proof.GetAttributes().GetAttributes() {
				proofAttrs[i] = didcomm.ProofAttribute{
					ID:        attribute.ID,
					Name:      attribute.Name,
					CredDefID: attribute.CredDefID,
				}
			}
			glog.V(3).Infoln("set proof from attrs")
		}

		// predicates - optional
		if proof.GetPredicatesJSON() != "" {
			dto.FromJSONStr(proof.GetPredicatesJSON(), &proofPredicates)
			glog.V(3).Infoln("set proof predicates from json:", proof.GetPredicatesJSON())
		} else if proof.GetPredicates() != nil {
			proofPredicates = make([]didcomm.ProofPredicate, len(proof.GetPredicates().GetPredicates()))
			for i, predicate := range proof.GetPredicates().GetPredicates() {
				proofPredicates[i] = didcomm.ProofPredicate{
					ID:     predicate.ID,
					Name:   predicate.Name,
					PType:  predicate.PType,
					PValue: predicate.PValue,
				}
			}
			glog.V(3).Infoln("set proof from predicates")
		}

		glog.V(1).Infof(
			"Create task for PresentProof with connection id %s, role %s",
			header.ConnID,
			protocol.GetRole().String(),
		)
	}

	return &taskPresentProof{
		TaskBase:        comm.TaskBase{TaskHeader: *header},
		ProofAttrs:      proofAttrs,
		ProofPredicates: proofPredicates,
	}, nil
}

func generateProofRequest(proofTask *taskPresentProof) *anoncreds.ProofRequest {
	reqAttrs := make(map[string]anoncreds.AttrInfo)
	for index, attr := range proofTask.ProofAttrs {
		restrictions := make([]anoncreds.Filter, 0)
		if attr.CredDefID != "" {
			restrictions = append(restrictions, anoncreds.Filter{CredDefID: attr.CredDefID})
		}
		id := "attr_referent_" + strconv.Itoa(index+1)
		if attr.ID != "" {
			id = attr.ID
		}
		reqAttrs[id] = anoncreds.AttrInfo{
			Name:         attr.Name,
			Restrictions: restrictions,
		}
	}
	reqPredicates := make(map[string]anoncreds.PredicateInfo)
	if proofTask.ProofPredicates != nil {
		for index, predicate := range proofTask.ProofPredicates {
			// TODO: restrictions
			id := "predicate_" + strconv.Itoa(index+1)
			if predicate.ID != "" {
				id = predicate.ID
			}
			reqPredicates[id] = anoncreds.PredicateInfo{
				Name:   predicate.Name,
				PType:  predicate.PType,
				PValue: int(predicate.PValue),
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

func startProofProtocol(ca comm.Receiver, t comm.Task) {
	defer err2.Catch()

	proofTask, ok := t.(*taskPresentProof)
	assert.That(ok)

	switch t.Type() {
	case pltype.CAProofPropose: // ----- prover will start -----
		try.To(prot.StartPSM(prot.Initial{
			SendNext:    pltype.PresentProofPropose,
			WaitingNext: pltype.PresentProofRequest,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) error {
				// Note!! StartPSM() sends certain Task fields to other end
				// as PL.Message like msg.ID, .SubMsg, .Info

				attrs := make([]presentproof.Attribute, len(proofTask.ProofAttrs))
				for i, attr := range proofTask.ProofAttrs {
					attrs[i] = presentproof.Attribute{
						Name:      attr.Name,
						CredDefID: attr.CredDefID,
					}
				}
				pp := presentproof.NewPreviewWithAttributes(attrs)

				propose := msg.FieldObj().(*presentproof.Propose)
				propose.PresentationProposal = pp
				propose.Comment = proofTask.Comment

				rep := &data.PresentProofRep{
					StateKey:   key,
					WeProposed: true,
				}
				return psm.AddRep(rep)
			},
		}))
	case pltype.CAProofRequest: // ----- verifier will start -----
		try.To(prot.StartPSM(prot.Initial{
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
				proofRequest := generateProofRequest(proofTask)
				// get proof req from task came in
				proofReqStr := dto.ToJSON(proofRequest)

				// set proof req to outgoing request message
				req := msg.FieldObj().(*presentproof.Request)
				req.RequestPresentations = presentproof.NewRequestPresentation(
					pltype.LibindyRequestPresentationID, []byte(proofReqStr))

				// create Rep and save it for PSM to run protocol
				rep := &data.PresentProofRep{
					StateKey: key,
					// Verifier cannot provide this..
					ProofReq: proofReqStr, //  .. but it gives this one.
				}
				return psm.AddRep(rep)
			},
		}))
	default:
		glog.Error("unsupported protocol start api type ")
	}
}

func continueProtocol(ca comm.Receiver, im didcomm.Msg) {
	defer err2.Catch()

	assert.That(im.Thread().ID != "", "continue present proof, packet thread ID missing")

	var continuators = map[string]continuatorFunc{
		pltype.SAPresentProofAcceptPropose: verifier.ContinueProposePresentation,
		pltype.SAPresentProofAcceptValues:  verifier.ContinueHandlePresentation,
		pltype.CANotifyUserAction:          prover.UserActionProofPresentation,
	}

	key := &psm.StateKey{
		DID:   ca.WDID(),
		Nonce: im.Thread().ID,
	}

	state := try.To1(psm.GetPSM(*key))
	assert.That(state != nil, "continue present proof, task not found")

	proofTask := state.LastState().T

	continuator, ok := continuators[proofTask.UserActionType()]
	if !ok {
		glog.Info(string(im.JSON()))
		s := "no continuator in present proof processor"
		glog.Error(s)
		panic(s)
	}
	continuator(ca, im)
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
		InOut: func(_ string, _, _ didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "proof ACK handler")
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
		InOut: func(_ string, _, _ didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "proof NACK handler")
			return false, nil
		},
	})
}

func fillPresentProofStatus(workerDID string, taskID string, ps *pb.ProtocolStatus) *pb.ProtocolStatus {
	defer err2.Catch(err2.Err(func(err error) {
		glog.Error("Failed to fill present proof status: ", err)
	}))

	assert.That(ps != nil)

	status := ps

	proofRep := try.To1(data.GetPresentProofRep(psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}))

	attrs := make([]*pb.Protocol_Proof_Attribute, 0, len(proofRep.Attributes))

	for _, attr := range proofRep.Attributes {
		a := &pb.Protocol_Proof_Attribute{
			Name:      attr.Name,
			CredDefID: attr.CredDefID,
			Value:     attr.Value,
		}
		attrs = append(attrs, a)
	}

	status.Status = &pb.ProtocolStatus_PresentProof{
		PresentProof: &pb.ProtocolStatus_PresentProofStatus{
			Proof: &pb.Protocol_Proof{
				Attributes: attrs,
			},
		},
	}

	return status
}
