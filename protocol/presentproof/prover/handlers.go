// Package prover includes Aries protocol handlers for a prover.
package prover

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/presentproof/preview"
	"github.com/findy-network/findy-agent/std/presentproof"
	"github.com/lainio/err2"
)

// HandleRequestPresentation is a handler func at PROVER side.
func HandleRequestPresentation(packet comm.Packet) (err error) {
	defer err2.Return(&err)

	key := psm.NewStateKey(packet.Receiver, packet.Payload.ThreadID())
	rep, _ := psm.GetPresentProofRep(key) // ignore not found error

	// If we* didn't start the Proof Protocol, we must ask user does she want
	// to reply i.e. present a proof, ___ *we == Prover

	if rep == nil || !rep.WeProposed { // we didn't start, verifier asked

		rep := &psm.PresentProofRep{
			Key:        key,
			WeProposed: false,
		}
		err2.Check(psm.AddPresentProofRep(rep))

		sendNext, waitingNext := checkAutoPermission(packet)

		return prot.ExecPSM(prot.Transition{
			Packet:      packet,
			SendNext:    sendNext,
			WaitingNext: waitingNext,
			InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
				defer err2.Annotate("proof req handler", &err)

				agent := packet.Receiver
				repK := psm.NewStateKey(agent, im.Thread().ID)
				rep := e2.PresentProofRep.Try(psm.GetPresentProofRep(repK))

				req := im.FieldObj().(*presentproof.Request)
				data := err2.Bytes.Try(presentproof.ProofReqData(req))
				rep.ProofReq = string(data)

				preview.StoreProofData(data, rep)

				pres, autoAccept := om.FieldObj().(*presentproof.Presentation)
				if autoAccept {
					err2.Check(rep.CreateProof(packet, repK.DID))
					pres.PresentationAttaches = presentproof.NewPresentationAttach(
						pltype.LibindyPresentationID, []byte(rep.Proof))
				}

				// Save the proof request to the Proof Rep
				err2.Check(psm.AddPresentProofRep(rep))

				return true, nil
			},
		})
	}
	// user has started the proof protocol herself, we can present a proof
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.PresentProofPresentation,
		WaitingNext: pltype.PresentProofACK,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("proof req handler", &err)

			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)
			rep := e2.PresentProofRep.Try(psm.GetPresentProofRep(repK))

			req := im.FieldObj().(*presentproof.Request)
			data := err2.Bytes.Try(presentproof.ProofReqData(req))
			rep.ProofReq = string(data)
			err2.Check(rep.CreateProof(packet, repK.DID))

			pres := om.FieldObj().(*presentproof.Presentation)
			pres.PresentationAttaches = presentproof.NewPresentationAttach(
				pltype.LibindyPresentationID, []byte(rep.Proof))

			// Save the proof request to the Proof Rep
			err2.Check(psm.AddPresentProofRep(rep))

			return true, nil
		},
	})
}

func checkAutoPermission(packet comm.Packet) (next string, wait string) {
	if packet.Receiver.AutoPermission() {
		next = pltype.PresentProofPresentation
		wait = pltype.PresentProofACK
	} else {
		next = pltype.Nothing
		wait = pltype.PresentProofUserAction
	}
	return next, wait
}
