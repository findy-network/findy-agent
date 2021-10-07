package issuecredential

import (
	"encoding/gob"
	"encoding/json"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/issuecredential/holder"
	"github.com/findy-network/findy-agent/protocol/issuecredential/issuer"
	"github.com/findy-network/findy-agent/std/issuecredential"
	"github.com/findy-network/findy-common-go/dto"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

type taskIssueCredential struct {
	comm.TaskBase
	Comment         string
	CredentialAttrs []didcomm.CredentialAttribute
	CredDefID       string
}

type statusIssueCredential struct {
	CredDefID  string                        `json:"credDefId"`
	SchemaID   string                        `json:"schemaId"`
	Attributes []didcomm.CredentialAttribute `json:"attributes"`
}

type handlerFunc func(packet comm.Packet, task comm.Task) (err error)
type continuatorFunc func(ca comm.Receiver, im didcomm.Msg)

var issueCredentialProcessor = comm.ProtProc{
	Creator:     createIssueCredentialTask,
	Starter:     startIssueCredentialByPropose,
	Continuator: continueProtocol,
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
	gob.Register(&taskIssueCredential{})
	prot.AddCreator(pltype.ProtocolIssueCredential, issueCredentialProcessor)
	prot.AddStarter(pltype.CACredRequest, issueCredentialProcessor)
	prot.AddStarter(pltype.CACredOffer, issueCredentialProcessor)
	prot.AddContinuator(pltype.CAContinueIssueCredentialProtocol, issueCredentialProcessor)
	prot.AddStatusProvider(pltype.ProtocolIssueCredential, issueCredentialProcessor)
	comm.Proc.Add(pltype.ProtocolIssueCredential, issueCredentialProcessor)
}

func createIssueCredentialTask(header *comm.TaskHeader, protocol *pb.Protocol) (t comm.Task, err error) {
	defer err2.Annotate("createIssueCredentialTask", &err)

	var credAttrs []didcomm.CredentialAttribute
	var credDefID string

	if protocol != nil {
		cred := protocol.GetIssueCredential()
		assert.P.True(cred != nil, "issue credential data missing")
		assert.P.True(
			protocol.GetRole() == pb.Protocol_INITIATOR || protocol.GetRole() == pb.Protocol_ADDRESSEE,
			"role is needed for issuing protocol")

		if cred.GetAttributesJSON() != "" {
			dto.FromJSONStr(cred.GetAttributesJSON(), &credAttrs)
			glog.V(3).Infoln("set cred attrs from json")
		} else {
			assert.P.True(cred.GetAttributes() != nil, "issue credential attributes data missing")
			credAttrs = make([]didcomm.CredentialAttribute, len(cred.GetAttributes().GetAttributes()))
			for i, attribute := range cred.GetAttributes().GetAttributes() {
				credAttrs[i] = didcomm.CredentialAttribute{
					Name:  attribute.Name,
					Value: attribute.Value,
				}
			}
			glog.V(3).Infoln("set cred from attrs")
		}
		glog.V(1).Infof(
			"Create task for IssueCredential with connection id %s, role %s",
			header.ConnID,
			protocol.GetRole().String(),
		)
		credDefID = cred.CredDefID
	}

	return &taskIssueCredential{
		TaskBase:        comm.TaskBase{TaskHeader: *header},
		CredentialAttrs: credAttrs,
		CredDefID:       credDefID,
	}, nil
}

// startIssueCredentialByPropose starts the Issue Credential Protocol by sending
// a Propose Message to pairwise identified by t.Message. It sends the protocol
// message from cloud EA, and saves the received credentials to cloud EA's
// wallet.
func startIssueCredentialByPropose(ca comm.Receiver, t comm.Task) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	credTask, ok := t.(*taskIssueCredential)
	assert.P.True(ok)

	switch t.Type() {
	case pltype.CACredOffer: // Send to Holder
		err2.Check(prot.StartPSM(prot.Initial{
			SendNext:    pltype.IssueCredentialOffer,
			WaitingNext: pltype.IssueCredentialRequest,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) (err error) {
				defer err2.Annotate("start issuing prot", &err)

				r := <-anoncreds.IssuerCreateCredentialOffer(
					ca.Wallet(), credTask.CredDefID)
				err2.Check(r.Err())
				credOffer := r.Str1()

				attrsStr := err2.Bytes.Try(json.Marshal(credTask.CredentialAttrs))
				pc := issuecredential.NewPreviewCredential(string(attrsStr))

				codedValues := issuecredential.PreviewCredentialToCodedValues(pc)
				rep := &psm.IssueCredRep{
					Key:        key,
					CredDefID:  credTask.CredDefID,
					Values:     codedValues,
					CredOffer:  credOffer,
					Attributes: credTask.CredentialAttrs,
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

				attrsStr, err := json.Marshal(credTask.CredentialAttrs)
				err2.Check(err)
				pc := issuecredential.NewPreviewCredential(string(attrsStr))

				propose := msg.FieldObj().(*issuecredential.Propose)
				propose.CredDefID = credTask.CredDefID
				propose.CredentialProposal = pc
				propose.Comment = credTask.Comment

				rep := &psm.IssueCredRep{
					Key:        key,
					CredDefID:  credTask.CredDefID,
					Attributes: credTask.CredentialAttrs,
					Values:     issuecredential.PreviewCredentialToCodedValues(pc),
				}
				err2.Check(psm.AddIssueCredRep(rep))
				return nil
			},
		}))
	}
}

// handleCredentialNACK is holder`s protocol function for now.
func handleCredentialNACK(packet comm.Packet) (err error) {
	return prot.ExecPSM(prot.Transition{
		Packet:      packet,
		SendNext:    pltype.Terminate, // this ends here
		WaitingNext: pltype.Terminate, // no next state
		InOut: func(connID string, im, om didcomm.MessageHdr) (ack bool, err error) {
			defer err2.Annotate("cred NACK", &err)
			// return false to mark this PSM to NACK!
			return false, nil
		},
	})
}

func continueProtocol(ca comm.Receiver, im didcomm.Msg) {
	defer err2.CatchTrace(func(err error) {
		glog.Error(err)
	})

	assert.D.True(im.Thread().ID != "", "continue issue credential, packet thread ID missing")

	var continuators = map[string]continuatorFunc{
		pltype.SAIssueCredentialAcceptPropose: issuer.ContinueCredentialPropose,
		pltype.CANotifyUserAction:             holder.UserActionCredential,
	}

	key := &psm.StateKey{
		DID:   ca.WDID(),
		Nonce: im.Thread().ID,
	}

	state, _ := psm.GetPSM(*key)
	assert.D.True(state != nil, "continue issue credential, task not found")

	credTask := state.LastState().T.(*taskIssueCredential)

	continuator, ok := continuators[credTask.UserActionType()]
	if !ok {
		glog.Info(string(im.JSON()))
		s := "no continuator in issue credential processor"
		glog.Error(s)
		panic(s)
	}
	continuator(ca, im)
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
