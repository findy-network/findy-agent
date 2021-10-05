package issuer

import (
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/issuecredential/preview"
	"github.com/findy-network/findy-agent/protocol/issuecredential/task"
	"github.com/findy-network/findy-agent/std/issuecredential"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

// HandleCredentialPropose is protocol function for IssueCredentialPropose at Issuer.
// Note! This is not called in the case where Issuer starts the protocol by
// sending Cred_Offer.
func HandleCredentialPropose(packet comm.Packet, credTask *task.TaskIssueCredential) (err error) {
	var sendNext, waitingNext string
	if packet.Receiver.AutoPermission() {
		sendNext = pltype.IssueCredentialOffer
		waitingNext = pltype.IssueCredentialRequest
	} else {
		sendNext = pltype.Nothing
		waitingNext = pltype.IssueCredentialUserAction
	}

	credTask.ActionType = task.AcceptPropose
	credTask.TaskBase.TaskHeader.UAType = pltype.SAIssueCredentialAcceptPropose

	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    sendNext,
		WaitingNext: waitingNext,
		SendOnNACK:  pltype.IssueCredentialNACK,
		Task:        credTask,
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("credential propose handler", &err)

			agent := packet.Receiver
			meDID := agent.Trans().MessagePipe().In.Did()

			prop := im.FieldObj().(*issuecredential.Propose)

			values := issuecredential.PreviewCredentialToCodedValues(
				prop.CredentialProposal)

			// TODO: support changing values

			r := <-anoncreds.IssuerCreateCredentialOffer(
				agent.MyCA().Wallet(), prop.CredDefID)
			err2.Check(r.Err())
			credOffer := r.Str1()

			rep := &psm.IssueCredRep{
				Key:       psm.StateKey{DID: meDID, Nonce: im.Thread().ID},
				CredDefID: prop.CredDefID,
				CredOffer: credOffer,
				Values:    values, // important! saved for Req handling
			}
			err2.Check(psm.AddIssueCredRep(rep))

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
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	err2.Check(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.IssueCredentialOffer,
		WaitingNext: pltype.IssueCredentialRequest,
		SendOnNACK:  pltype.IssueCredentialNACK,
		Transfer: func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("credential propose user action handler", &err)

			iMsg := im.(didcomm.Msg)
			ack = iMsg.Ready()
			if !ack {
				glog.Warning("user doesn't accept the cred propose")
				return ack, nil
			}

			repK := psm.NewStateKey(ca, im.Thread().ID)

			rep := e2.IssueCredRep.Try(psm.GetIssueCredRep(repK))

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
func HandleCredentialRequest(packet comm.Packet, credTask *task.TaskIssueCredential) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.IssueCredentialIssue,
		WaitingNext: pltype.IssueCredentialACK,
		Task:        credTask,
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
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
func HandleCredentialACK(packet comm.Packet, credTask *task.TaskIssueCredential) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate, // this ends here
		WaitingNext: pltype.Terminate, // no next state
		Task:        credTask,
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("cred ACK", &err)
			return true, nil
		},
	})
}
