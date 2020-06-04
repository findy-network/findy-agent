package issuer

import (
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/e2"
	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/prot"
	"github.com/optechlab/findy-agent/agent/psm"
	"github.com/optechlab/findy-agent/protocol/issuecredential/preview"
	"github.com/optechlab/findy-agent/std/issuecredential"
	"github.com/optechlab/findy-go/anoncreds"
)

// HandleCredentialPropose is protocol function for IssueCredentialPropose at Issuer.
// Note! This is not called in the case where Issuer starts the protocol by
// sending Cred_Offer.
func HandleCredentialPropose(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.IssueCredentialOffer,
		WaitingNext: pltype.IssueCredentialRequest,
		SendOnNACK:  pltype.IssueCredentialNACK,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("credential propose handler", &err)

			agent := packet.Receiver
			meDID := agent.Trans().MessagePipe().In.Did()

			prop := im.FieldObj().(*issuecredential.Propose)

			// we calculate attr values here even it's not necessary
			// because the SA will usually update them by it self.
			values := issuecredential.PreviewCredentialToCodedValues(
				prop.CredentialProposal)

			var subMsg map[string]interface{}
			if im.Thread().PID != "" {
				subMsg = make(map[string]interface{}, 1)
				subMsg["id"] = im.Thread().PID
			}
			saMsg := mesg.MsgCreator.Create(didcomm.MsgInit{
				Nonce: im.Thread().ID,
				ID:    prop.CredDefID,
				Info:  values,
				Msg:   subMsg,
			}).(didcomm.Msg)
			eaAnswer := e2.M.Try(agent.MyCA().CallEA(
				pltype.SAIssueCredentialAcceptPropose, saMsg))
			if !eaAnswer.Ready() { // if EA wont accept
				glog.Warning("EA rejects API call:", eaAnswer.Error())
				return false, nil
			}
			if eaAnswer.Info() != "" && eaAnswer.Info() != values {
				glog.V(1).Info("SA sets the cred attrs")
				values = eaAnswer.Info()
			}

			r := <-anoncreds.IssuerCreateCredentialOffer(
				agent.MyCA().Wallet(), prop.CredDefID)
			err2.Check(r.Err())
			credOffer := r.Str1()

			offer := om.FieldObj().(*issuecredential.Offer)
			offer.OffersAttach =
				issuecredential.NewOfferAttach([]byte(credOffer))
			offer.CredentialPreview =
				issuecredential.NewPreviewCredentialRaw(values)
			offer.Comment = values // todo: for legacy tests

			rep := &psm.IssueCredRep{
				Key:       psm.StateKey{DID: meDID, Nonce: im.Thread().ID},
				CredDefID: prop.CredDefID,
				CredOffer: credOffer,
				Values:    values, // important! saved for Req handling
			}
			preview.StoreCredPreview(&offer.CredentialPreview, rep)
			err2.Check(psm.AddIssueCredRep(rep))

			return true, nil
		},
	})
}

// HandleCredentialRequest implements the handler for credential request protocol
// msg. This is Issuer side action.
func HandleCredentialRequest(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.IssueCredentialIssue,
		WaitingNext: pltype.IssueCredentialACK,
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("cred req", &err)

			req := im.FieldObj().(*issuecredential.Request)
			agent := packet.Receiver
			repK := psm.NewStateKey(agent, im.Thread().ID)

			rep := e2.IssueCredRep.Try(psm.GetIssueCredRep(repK))
			attach := err2.Bytes.Try(issuecredential.RequestAttach(req))
			credReq := string(attach)
			cred := err2.String.Try(rep.IssuerBuildCred(packet, credReq))

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
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("cred ACK", &err)
			return true, nil
		},
	})
}
