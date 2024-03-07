package issuer

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/issuecredential/data"
	"github.com/findy-network/findy-agent/protocol/issuecredential/preview"
	"github.com/findy-network/findy-agent/std/issuecredential"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

// HandleCredentialPropose is protocol function for IssueCredentialPropose at Issuer.
// Note! This is not called in the case where Issuer starts the protocol by
// sending Cred_Offer.
func HandleCredentialPropose(packet comm.Packet) (err error) {
	var sendNext, waitingNext string
	if packet.Receiver.AutoPermission() {
		sendNext = pltype.IssueCredentialOffer
		waitingNext = pltype.IssueCredentialRequest
	} else {
		sendNext = pltype.Nothing
		waitingNext = pltype.IssueCredentialUserAction
	}

	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    sendNext,
		WaitingNext: waitingNext,
		SendOnNACK:  pltype.IssueCredentialNACK,
		TaskHeader:  &comm.TaskHeader{UserActionPLType: pltype.SAIssueCredentialAcceptPropose},
		InOut: func(_ string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "credential propose handler")

			wa := packet.Receiver
			meDID := wa.MyDID().Did()

			prop := im.FieldObj().(*issuecredential.Propose)

			values := issuecredential.PreviewCredentialToCodedValues(
				prop.CredentialProposal)

			attributes := make([]didcomm.CredentialAttribute, 0)
			for _, attr := range prop.CredentialProposal.Attributes {
				attributes = append(attributes, didcomm.CredentialAttribute{
					Name:     attr.Name,
					Value:    attr.Value,
					MimeType: attr.MimeType,
				})
			}

			// TODO: support changing values

			r := <-anoncreds.IssuerCreateCredentialOffer(
				wa.Wallet(), prop.CredDefID)
			try.To(r.Err())
			credOffer := r.Str1()

			rep := &data.IssueCredRep{
				StateKey:   psm.StateKey{DID: meDID, Nonce: im.Thread().ID},
				CredDefID:  prop.CredDefID,
				CredOffer:  credOffer,
				Values:     values, // important! saved for Req handling
				Attributes: attributes,
			}
			try.To(psm.AddRep(rep))

			offer, autoAccept := om.FieldObj().(*issuecredential.Offer)
			if autoAccept {
				offer.OffersAttach =
					issuecredential.NewOfferAttach([]byte(credOffer))
				offer.CredentialPreview =
					issuecredential.NewPreviewCredentialRaw(values)
				offer.Comment = values // todo: for legacy tests
				preview.StoreCredPreview(&offer.CredentialPreview, rep)
			}

			return true, nil
		},
	})
}

// todo lapi: im message is old legacy api type!!
func ContinueCredentialPropose(ca comm.Receiver, im didcomm.Msg) {
	defer err2.Catch()

	try.To(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.IssueCredentialOffer,
		WaitingNext: pltype.IssueCredentialRequest,
		SendOnNACK:  pltype.IssueCredentialNACK,
		Transfer: func(_ comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "credential propose user action handler")

			iMsg := im.(didcomm.Msg)
			ack = iMsg.Ready()
			if !ack {
				glog.Warning("user doesn't accept the cred propose")
				return ack, nil
			}

			repK := psm.NewStateKey(ca, im.Thread().ID)

			rep := try.To1(data.GetIssueCredRep(repK))

			offer := om.FieldObj().(*issuecredential.Offer)
			offer.OffersAttach =
				issuecredential.NewOfferAttach([]byte(rep.CredOffer))
			offer.CredentialPreview =
				issuecredential.NewPreviewCredentialRaw(rep.Values)
			offer.Comment = rep.Values // todo: for legacy tests
			preview.StoreCredPreview(&offer.CredentialPreview, rep)

			return true, nil
		},
	}))
}

// HandleCredentialRequest implements the handler for credential request protocol
// msg. This is Issuer side action.
func HandleCredentialRequest(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.IssueCredentialIssue,
		WaitingNext: pltype.IssueCredentialACK,
		InOut: func(_ string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "cred req")

			req := im.FieldObj().(*issuecredential.Request)
			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)

			rep := try.To1(data.GetIssueCredRep(repK))
			attach := try.To1(issuecredential.RequestAttach(req))
			credReq := string(attach)
			cred := try.To1(rep.IssuerBuildCred(packet, credReq))

			issue := om.FieldObj().(*issuecredential.Issue)
			issue.CredentialsAttach =
				issuecredential.NewCredentialsAttach([]byte(cred))

			return true, nil
		},
	})
}

// HandleCredentialACK is Issuer's protocol function. This message is currently
// received at issuer's side. However, in future implementations we might move
// it into the processor.
func HandleCredentialACK(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate, // this ends here
		WaitingNext: pltype.Terminate, // no next state
		InOut: func(_ string, _, _ didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Handle(&err, "cred ACK")
			return true, nil
		},
	})
}
