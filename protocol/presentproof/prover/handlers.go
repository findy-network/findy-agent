// Package prover includes Aries protocol handlers for a prover.
package prover

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/presentproof/data"
	"github.com/findy-network/findy-agent/protocol/presentproof/preview"
	"github.com/findy-network/findy-agent/std/presentproof"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

// HandleRequestPresentation is a handler func at PROVER side.
func HandleRequestPresentation(packet comm.Packet) (err error) {
	defer err2.Handle(&err)

	key := psm.NewStateKey(packet.Receiver, packet.Payload.ThreadID())
	rep, _ := data.GetPresentProofRep(key) // ignore not found error

	if rep == nil {
		rep := &data.PresentProofRep{
			StateKey:   key,
			WeProposed: false,
		}
		try.To(psm.AddRep(rep))
	}

	sendNext, waitingNext := checkAutoPermission(packet)

	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    sendNext,
		WaitingNext: waitingNext,
		TaskHeader:  &comm.TaskHeader{UserActionPLType: pltype.CANotifyUserAction},
		InOut: func(_ string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "proof req handler")

			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)
			rep := try.To1(data.GetPresentProofRep(repK))

			req := im.FieldObj().(*presentproof.Request)
			data := try.To1(presentproof.ProofReqData(req))
			rep.ProofReq = string(data)

			preview.StoreProofData(data, rep)

			pres, autoAccept := om.FieldObj().(*presentproof.Presentation)
			if autoAccept {
				try.To(rep.CreateProof(packet, repK.DID))
				pres.PresentationAttaches = presentproof.NewPresentationAttach(
					pltype.LibindyPresentationID, []byte(rep.Proof))
			}

			// Save the proof request to the Proof Rep
			try.To(psm.AddRep(rep))

			return true, nil
		},
	})
}

func UserActionProofPresentation(ca comm.Receiver, im didcomm.Msg) {
	defer err2.Catch()

	try.To(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.PresentProofPresentation,
		WaitingNext: pltype.PresentProofACK,
		SendOnNACK:  pltype.PresentProofNACK,
		Transfer: func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "proof user action handler")

			// Does user allow continue?
			iMsg := im.(didcomm.Msg)
			if !iMsg.Ready() {
				glog.Warning("user doesn't accept proof")
				return false, nil
			}

			// We continue, get previous data, create the proof and send it
			agent := wa
			repK := psm.NewStateKey(agent, im.Thread().ID)
			rep := try.To1(data.GetPresentProofRep(repK))

			try.To(rep.CreateProof(comm.Packet{Receiver: agent}, repK.DID))
			// save created proof to Representative
			try.To(psm.AddRep(rep))

			pres := om.FieldObj().(*presentproof.Presentation)
			pres.PresentationAttaches = presentproof.NewPresentationAttach(
				pltype.LibindyPresentationID, []byte(rep.Proof))

			return true, nil
		},
	}))
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
