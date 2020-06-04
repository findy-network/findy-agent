package holder

import (
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/e2"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/prot"
	"github.com/optechlab/findy-agent/agent/psm"
	"github.com/optechlab/findy-agent/protocol/issuecredential/preview"
	"github.com/optechlab/findy-agent/std/common"
	"github.com/optechlab/findy-agent/std/issuecredential"
	"github.com/optechlab/findy-go/dto"
)

// HandleCredentialOffer is protocol function for CRED_OFF at prover/holder
func HandleCredentialOffer(packet comm.Packet) (err error) {
	defer err2.Return(&err)

	// First check who is starting the protocol. If we receive this as a first
	// message, other end (SA) is offering a cred for us. Otherwise we have
	// already started the protocol by sending issue cred proposal.

	// Make a PSM key and find protocol representative Rep
	key := psm.NewStateKey(packet.Receiver, packet.Payload.ThreadID())
	rep, _ := psm.GetIssueCredRep(key) // if rep not found, ignore error

	// If we didn't start the Issuing protocol, we must ask user does she want
	// to reply i.e. send a cred request

	// Do we have a Rep?
	if rep == nil {
		rep := &psm.IssueCredRep{
			Key: key,
		}
		err2.Check(psm.AddIssueCredRep(rep))

		return prot.ExecPSM(prot.Transition{
			Packet:      packet,
			SendNext:    pltype.Nothing,
			WaitingNext: pltype.IssueCredentialUserAction,
			InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
				defer err2.Annotate("cred offer ask user", &err)

				offer := im.FieldObj().(*issuecredential.Offer)
				values := issuecredential.PreviewCredentialToValues(
					offer.CredentialPreview)

				agent := packet.Receiver
				repK := psm.NewStateKey(agent, im.Thread().ID)
				rep := e2.IssueCredRep.Try(psm.GetIssueCredRep(repK))

				attach := err2.Bytes.Try(issuecredential.OfferAttach(offer))
				rep.CredOffer = string(attach)

				// we need to parse the cred_def_id from credOffer
				var subMsg map[string]interface{}
				dto.FromJSON(attach, &subMsg)
				if credDefID, ok := subMsg["cred_def_id"]; ok {
					rep.CredDefID = credDefID.(string)
				}
				rep.Values = values
				preview.StoreCredPreview(&offer.CredentialPreview, rep)
				// Save the Rep with the offer
				err2.Check(psm.AddIssueCredRep(rep))

				return true, nil
			},
		})
	}

	// So, user has started the proof protocol herself, she can just continue
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.IssueCredentialRequest,
		WaitingNext: pltype.IssueCredentialIssue,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("cred offer", &err)

			offer := im.FieldObj().(*issuecredential.Offer)
			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)

			rep := e2.IssueCredRep.Try(psm.GetIssueCredRep(repK))
			attach := err2.Bytes.Try(issuecredential.OfferAttach(offer))
			rep.CredOffer = string(attach)
			rep.Values = issuecredential.PreviewCredentialToValues(
				offer.CredentialPreview)
			preview.StoreCredPreview(&offer.CredentialPreview, rep)

			credRq := err2.String.Try(rep.BuildCredRequest(packet))
			err2.Check(psm.AddIssueCredRep(rep))

			req := om.FieldObj().(*issuecredential.Request)
			req.RequestsAttach =
				issuecredential.NewRequestAttach([]byte(credRq))

			return true, nil
		},
	})
}

// HandleCredentialIssue is protocol function for CRED_ISSUE for prover/holder.
func HandleCredentialIssue(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.IssueCredentialACK,
		WaitingNext: pltype.Terminate, // no next state, we are fine
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("cred issue", &err)

			issue := im.FieldObj().(*issuecredential.Issue)
			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)

			rep := e2.IssueCredRep.Try(psm.GetIssueCredRep(repK))
			cred := err2.Bytes.Try(issuecredential.CredentialAttach(issue))
			err2.Check(rep.StoreCred(packet, string(cred)))

			outAck := om.FieldObj().(*common.Ack)
			outAck.Status = "OK"

			return true, nil
		},
	})
}
