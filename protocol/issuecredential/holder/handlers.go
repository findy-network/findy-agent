package holder

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/issuecredential/data"
	"github.com/findy-network/findy-agent/protocol/issuecredential/preview"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/issuecredential"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

// HandleCredentialOffer is protocol function for CRED_OFF at prover/holder
func HandleCredentialOffer(packet comm.Packet) (err error) {
	defer err2.Handle(&err)

	// First check who is starting the protocol. If we receive this as a first
	// message, other end (SA) is offering a cred for us. Otherwise we have
	// already started the protocol by sending issue cred proposal.

	// Make a PSM key and find protocol representative Rep
	key := psm.NewStateKey(packet.Receiver, packet.Payload.ThreadID())
	rep, _ := data.GetIssueCredRep(key) // if rep not found, ignore error

	// If we didn't start the Issuing protocol, we must ask user does she want
	// to reply i.e. send a cred request

	// Do we have a Rep?
	if rep == nil {
		rep := &data.IssueCredRep{
			StateKey: key,
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
			defer err2.Handle(&err, "cred offer ask user (%v)",
				packet.Receiver.RootDid().Did())

			offer := im.FieldObj().(*issuecredential.Offer)
			values := issuecredential.PreviewCredentialToValues(
				offer.CredentialPreview)

			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)
			rep := try.To1(data.GetIssueCredRep(repK))

			attach := try.To1(issuecredential.OfferAttach(offer))
			rep.CredOffer = string(attach)

			// we need to parse the cred_def_id from credOffer
			var subMsg map[string]interface{}
			dto.FromJSON(attach, &subMsg)
			if credDefID, ok := subMsg["cred_def_id"]; ok {
				rep.CredDefID = credDefID.(string)
			}
			defer err2.Handle(&err, "cred def (%v)", rep.CredDefID)

			rep.Values = values
			preview.StoreCredPreview(&offer.CredentialPreview, rep)

			req, autoAccept := om.FieldObj().(*issuecredential.Request)
			if autoAccept {
				credRq := try.To1(rep.BuildCredRequest(packet))
				req.RequestsAttach =
					issuecredential.NewRequestAttach([]byte(credRq))
			}

			// Save the rep with the offer and with the request if
			// auto accept
			try.To(psm.AddRep(rep))

			return true, nil
		},
	})
}

// todo lapi: im message is old legacy api type!!

// userActionCredential is called when Holder has received a Cred_Offer and it's
// transferred the question to user: if she accepts the credential.
func UserActionCredential(ca comm.Receiver, im didcomm.Msg) {
	defer err2.Catch()

	try.To(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.IssueCredentialRequest,
		WaitingNext: pltype.IssueCredentialACK,
		SendOnNACK:  pltype.IssueCredentialNACK,
		Transfer: func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "issuing user action handler")

			iMsg := im.(didcomm.Msg)
			ack = iMsg.Ready()
			if !ack {
				glog.Warning("user doesn't accept the issuing")
				return ack, nil
			}

			agent := wa
			repK := psm.NewStateKey(agent, im.Thread().ID)

			rep := try.To1(data.GetIssueCredRep(repK))
			credRq := try.To1(rep.BuildCredRequest(
				comm.Packet{Receiver: agent}))

			try.To(psm.AddRep(rep))
			req := om.FieldObj().(*issuecredential.Request)
			req.RequestsAttach =
				issuecredential.NewRequestAttach([]byte(credRq))

			return true, nil
		},
	}))
}

func checkAutoPermission(packet comm.Packet) (next string, wait string) {
	if packet.Receiver.AutoPermission() {
		next = pltype.IssueCredentialRequest
		wait = pltype.IssueCredentialIssue
	} else {
		next = pltype.Nothing
		wait = pltype.IssueCredentialUserAction
	}
	return next, wait
}

// HandleCredentialIssue is protocol function for CRED_ISSUE for prover/holder.
func HandleCredentialIssue(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.IssueCredentialACK,
		WaitingNext: pltype.Terminate, // no next state, we are fine

		InOut: func(_ string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "cred issue")

			issue := im.FieldObj().(*issuecredential.Issue)
			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)

			rep := try.To1(data.GetIssueCredRep(repK))
			cred := try.To1(issuecredential.CredentialAttach(issue))
			try.To(rep.StoreCred(packet, string(cred)))

			outAck := om.FieldObj().(*common.Ack)
			outAck.Status = "OK"

			return true, nil
		},
	})
}
