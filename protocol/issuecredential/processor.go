package issuecredential

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/issuecredential/holder"
	"github.com/findy-network/findy-agent/protocol/issuecredential/issuer"
	"github.com/findy-network/findy-agent/std/issuecredential"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
)

type statusIssueCredential struct {
	CredDefID  string                        `json:"credDefId"`
	SchemaID   string                        `json:"schemaId"`
	Attributes []didcomm.CredentialAttribute `json:"attributes"`
}

var issueCredentialProcessor = comm.ProtProc{
	Starter:     startIssueCredentialByPropose,
	Continuator: userActionCredential,
	Handlers: map[string]comm.HandlerFunc{
		pltype.HandlerIssueCredentialPropose: issuer.HandleCredentialPropose,
		pltype.HandlerIssueCredentialOffer:   holder.HandleCredentialOffer,
		pltype.HandlerIssueCredentialRequest: issuer.HandleCredentialRequest,
		pltype.HandlerIssueCredentialIssue:   holder.HandleCredentialIssue,
		pltype.HandlerIssueCredentialACK:     issuer.HandleCredentialACK,
		pltype.HandlerIssueCredentialNACK:    handleCredentialNACK,
	},
	Status: getIssueCredentialStatus,
}

func init() {
	prot.AddStarter(pltype.CACredRequest, issueCredentialProcessor)
	prot.AddStarter(pltype.CACredOffer, issueCredentialProcessor)
	prot.AddContinuator(pltype.CAContinueIssueCredentialProtocol, issueCredentialProcessor)
	prot.AddStatusProvider(pltype.ProtocolIssueCredential, issueCredentialProcessor)
	comm.Proc.Add(pltype.ProtocolIssueCredential, issueCredentialProcessor)
}

// startIssueCredentialByPropose starts the Issue Credential Protocol by sending
// a Propose Message to pairwise identified by t.Message. It sends the protocol
// message from cloud EA, and saves the received credentials to cloud EA's
// wallet.
func startIssueCredentialByPropose(ca comm.Receiver, t *comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	switch t.TypeID {
	case pltype.CACredOffer: // Send to Holder
		err2.Check(prot.StartPSM(prot.Initial{
			SendNext:    pltype.IssueCredentialOffer,
			WaitingNext: pltype.IssueCredentialRequest,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) (err error) {
				defer err2.Annotate("start issuing prot", &err)

				r := <-anoncreds.IssuerCreateCredentialOffer(
					ca.Wallet(), *t.CredDefID)
				err2.Check(r.Err())
				credOffer := r.Str1()

				attrsStr := err2.Bytes.Try(json.Marshal(t.CredentialAttrs))
				pc := issuecredential.NewPreviewCredential(string(attrsStr))

				codedValues := issuecredential.PreviewCredentialToCodedValues(pc)
				rep := &psm.IssueCredRep{
					Key:        key,
					CredDefID:  *t.CredDefID,
					Values:     codedValues,
					CredOffer:  credOffer,
					Attributes: *t.CredentialAttrs,
				}
				err2.Check(psm.AddIssueCredRep(rep))

				offer := msg.FieldObj().(*issuecredential.Offer)
				offer.CredentialPreview = pc
				offer.OffersAttach = // here we send the indy cred offer
					issuecredential.NewOfferAttach([]byte(credOffer))

				return nil
			},
		}))

	case pltype.CACredRequest: // Send to Issuer
		err2.Check(prot.StartPSM(prot.Initial{
			SendNext:    pltype.IssueCredentialPropose,
			WaitingNext: pltype.IssueCredentialOffer,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) (err error) {
				defer err2.Annotate("start issue prot", &err)

				attrsStr, err := json.Marshal(t.CredentialAttrs)
				err2.Check(err)
				pc := issuecredential.NewPreviewCredential(string(attrsStr))

				propose := msg.FieldObj().(*issuecredential.Propose)
				propose.CredDefID = *t.CredDefID
				propose.CredentialProposal = pc
				propose.Comment = t.Info // todo: for legacy tests

				// here we have Session and verify ID, this is to make libindy work
				if id, ok := t.Msg["id"]; ok { // todo: legacy stuff!
					propose.Thread.PID = id.(string) //  take safe still
				}

				rep := &psm.IssueCredRep{
					Key:        key,
					CredDefID:  *t.CredDefID,
					Attributes: *t.CredentialAttrs,
					Values:     issuecredential.PreviewCredentialToCodedValues(pc),
				}
				err2.Check(psm.AddIssueCredRep(rep))
				return nil
			},
		}))
	}
}

// userActionCredential is called when Holder has received a Cred_Offer and it's
// transferred the question to user: if she accepts the credential.
func userActionCredential(ca comm.Receiver, im didcomm.Msg) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	err2.Check(prot.ContinuePSM(prot.Again{
		CA:          ca,
		InMsg:       im,
		SendNext:    pltype.IssueCredentialRequest,
		WaitingNext: pltype.IssueCredentialACK,
		SendOnNACK:  pltype.IssueCredentialNACK,
		Transfer: func(wa comm.Receiver, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("issuing user action handler", &err)

			iMsg := im.(didcomm.Msg)
			ack = iMsg.Ready()
			if !ack {
				glog.Warning("user doesn't accept the issuing")
				return ack, nil
			}

			agent := wa
			repK := psm.NewStateKey(agent, im.Thread().ID)

			rep := e2.IssueCredRep.Try(psm.GetIssueCredRep(repK))
			credRq := err2.String.Try(rep.BuildCredRequest(
				comm.Packet{Receiver: agent}))

			err2.Check(psm.AddIssueCredRep(rep))
			req := om.FieldObj().(*issuecredential.Request)
			req.RequestsAttach =
				issuecredential.NewRequestAttach([]byte(credRq))

			return true, nil
		},
	}))
}

// handleCredentialNACK is holder`s protocol function for now.
func handleCredentialNACK(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate, // this ends here
		WaitingNext: pltype.Terminate, // no next state
		InOut: func(im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("cred NACK", &err)
			// return false to mark this PSM to NACK!
			return false, nil
		},
	})
}

func getIssueCredentialStatus(workerDID string, taskID string) interface{} {
	defer err2.CatchTrace(func(err error) {
		glog.Error("Failed to set issue credential status: ", err)
	})
	key := &psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}

	credRep := e2.IssueCredRep.Try(psm.GetIssueCredRep(*key))

	// TODO: save schema id parsed to db?
	var credOffer interface{}
	err := json.Unmarshal([]byte(credRep.CredOffer), &credOffer)
	err2.Check(err)

	credOfferMap := credOffer.(map[string]interface{})
	schemaID := credOfferMap["schema_id"].(string)

	return statusIssueCredential{
		CredDefID:  credRep.CredDefID,
		SchemaID:   schemaID,
		Attributes: credRep.Attributes,
	}
}
