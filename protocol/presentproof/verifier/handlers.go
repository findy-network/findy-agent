// Package verifier includes Aries protocol handlers for a verifier.
package verifier

import (
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/protocol/presentproof/preview"
	"github.com/findy-network/findy-agent/std/presentproof"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
)

// HandleProposePresentation is a protocol handler function at VERIFIER side.
func HandleProposePresentation(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.PresentProofRequest,
		WaitingNext: pltype.PresentProofPresentation,
		SendOnNACK:  pltype.PresentProofNACK,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof propose handler", &err)

			agent := packet.Receiver
			meDID := agent.Trans().MessagePipe().In.Did()

			// Let SA EA to check if it's OK to start present proof
			saMsg := mesg.MsgCreator.Create(didcomm.MsgInit{
				Nonce: im.Thread().ID,
				ID:    im.Thread().PID,
			}).(didcomm.Msg)

			ca := agent.MyCA()
			eaAnswer := e2.M.Try(ca.CallEA(pltype.SAPresentProofAcceptPropose, saMsg))
			if !eaAnswer.Ready() { // if EA wont accept
				glog.Warning("EA rejects API call: ", eaAnswer.Error())
				return false, nil
			}

			// SA accepts and gives the proof req to send to the other end
			reqStr := dto.ToJSON(eaAnswer.SubMsg())
			// Save it ..
			rep := &psm.PresentProofRep{
				Key:      psm.StateKey{DID: meDID, Nonce: im.Thread().ID},
				ProofReq: reqStr,
			}
			err2.Check(psm.AddPresentProofRep(rep))

			// .. and send to a verifier
			req := om.FieldObj().(*presentproof.Request) // query interface
			req.RequestPresentations = presentproof.NewRequestPresentation(
				utils.UUID(), []byte(reqStr))

			return true, nil
		},
	})
}

// HandlePresentation is a protocol handler function at VERIFIER side for handling
// proof presentation.
func HandlePresentation(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.PresentProofACK,
		WaitingNext: pltype.Terminate,
		SendOnNACK:  pltype.PresentProofNACK,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof presentation handler", &err)

			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)
			rep := e2.PresentProofRep.Try(psm.GetPresentProofRep(repK))

			// 1st, verify the proof by our selves
			pres := im.FieldObj().(*presentproof.Presentation)
			data := err2.Bytes.Try(presentproof.Proof(pres))
			rep.Proof = string(data)

			if !err2.Bool.Try(rep.VerifyProof(packet)) {
				glog.Errorf("Cannot verify proof (nonce:%v) terminating presentation protocol", im.Thread().ID)
				return false, nil
			}

			preview.StoreProofData([]byte(rep.ProofReq), rep)
			err2.Check(psm.AddPresentProofRep(rep))

			var proof anoncreds.Proof
			dto.FromJSON(data, &proof)
			proofValues := make([]didcomm.ProofValue, len(rep.Attributes))
			for index, attr := range rep.Attributes {
				proofValues[index] =
					didcomm.ProofValue{
						Value:     proof.RequestedProof.RevealedAttrs[attr.ID].Raw,
						Name:      attr.Name,
						CredDefID: attr.CredDefID,
						Predicate: attr.Predicate,
					}
			}

			// .. then let SA EA check the values of the proof
			saMsg := mesg.MsgCreator.Create(didcomm.MsgInit{
				Nonce:       im.Thread().ID,
				ID:          im.Thread().PID,
				ProofValues: &proofValues,
			}).(didcomm.Msg)

			ca := agent.MyCA()
			eaAnswer := e2.M.Try(ca.CallEA(pltype.SAPresentProofAcceptValues, saMsg))
			if !eaAnswer.Ready() { // if EA wont accept
				glog.Warning("EA rejects API call: ", eaAnswer.Error())
				return false, nil
			}
			// All checks done, let's send ACK
			return true, nil
		},
	})
}
