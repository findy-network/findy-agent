package issuecredential

import (
	"encoding/gob"
	"encoding/json"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/protocol/issuecredential/data"
	"github.com/findy-network/findy-agent/protocol/issuecredential/holder"
	"github.com/findy-network/findy-agent/protocol/issuecredential/issuer"
	"github.com/findy-network/findy-agent/std/issuecredential"
	"github.com/findy-network/findy-common-go/dto"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type taskIssueCredential struct {
	comm.TaskBase
	Comment         string
	CredentialAttrs []didcomm.CredentialAttribute
	CredDefID       string
}

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
	FillStatus: fillIssueCredentialStatus,
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
	defer err2.Handle(&err, "createIssueCredentialTask")

	var credAttrs []didcomm.CredentialAttribute
	var credDefID string

	if protocol != nil {
		cred := protocol.GetIssueCredential()
		assert.That(cred != nil, "issue credential data missing")
		assert.That(
			protocol.GetRole() == pb.Protocol_INITIATOR || protocol.GetRole() == pb.Protocol_ADDRESSEE,
			"role is needed for issuing protocol")

		if cred.GetAttributesJSON() != "" {
			dto.FromJSONStr(cred.GetAttributesJSON(), &credAttrs)
			glog.V(3).Infoln("set cred attrs from json")
		} else {
			assert.That(cred.GetAttributes() != nil, "issue credential attributes data missing")
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
	defer err2.Catch()

	credTask, ok := t.(*taskIssueCredential)
	assert.That(ok)

	// ensure that mime type is set - some agent implementations are depending on it
	for index, attr := range credTask.CredentialAttrs {
		if attr.MimeType == "" {
			credTask.CredentialAttrs[index].MimeType = "text/plain"
		}
	}

	switch t.Type() {
	case pltype.CACredOffer: // Send to Holder
		try.To(prot.StartPSM(prot.Initial{
			SendNext:    pltype.IssueCredentialOffer,
			WaitingNext: pltype.IssueCredentialRequest,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) (err error) {
				defer err2.Handle(&err, "start issuing prot")

				r := <-anoncreds.IssuerCreateCredentialOffer(
					ca.WorkerEA().Wallet(), credTask.CredDefID)
				try.To(r.Err())
				credOffer := r.Str1()

				attrsStr := try.To1(json.Marshal(credTask.CredentialAttrs))
				pc := issuecredential.NewPreviewCredential(string(attrsStr))

				codedValues := issuecredential.PreviewCredentialToCodedValues(pc)
				rep := &data.IssueCredRep{
					StateKey:   key,
					CredDefID:  credTask.CredDefID,
					Values:     codedValues,
					CredOffer:  credOffer,
					Attributes: credTask.CredentialAttrs,
				}
				try.To(psm.AddRep(rep))

				offer := msg.FieldObj().(*issuecredential.Offer)
				offer.CredentialPreview = pc
				offer.OffersAttach = // here we send the indy cred offer
					issuecredential.NewOfferAttach([]byte(credOffer))

				return nil
			},
		}))

	case pltype.CACredRequest: // Send to Issuer
		try.To(prot.StartPSM(prot.Initial{
			SendNext:    pltype.IssueCredentialPropose,
			WaitingNext: pltype.IssueCredentialOffer,
			Ca:          ca,
			T:           t,
			Setup: func(key psm.StateKey, msg didcomm.MessageHdr) (err error) {
				defer err2.Handle(&err, "start issue prot")

				attrsStr := try.To1(json.Marshal(credTask.CredentialAttrs))
				pc := issuecredential.NewPreviewCredential(string(attrsStr))

				propose := msg.FieldObj().(*issuecredential.Propose)
				propose.CredDefID = credTask.CredDefID
				propose.CredentialProposal = pc
				propose.Comment = credTask.Comment

				rep := &data.IssueCredRep{
					StateKey:   key,
					CredDefID:  credTask.CredDefID,
					Attributes: credTask.CredentialAttrs,
					Values:     issuecredential.PreviewCredentialToCodedValues(pc),
				}
				try.To(psm.AddRep(rep))
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
			defer err2.Handle(&err, "cred NACK")
			// return false to mark this PSM to NACK!
			return false, nil
		},
	})
}

func continueProtocol(ca comm.Receiver, im didcomm.Msg) {
	defer err2.Catch()

	assert.That(im.Thread().ID != "", "continue issue credential, packet thread ID missing")

	var continuators = map[string]continuatorFunc{
		pltype.SAIssueCredentialAcceptPropose: issuer.ContinueCredentialPropose,
		pltype.CANotifyUserAction:             holder.UserActionCredential,
	}

	key := psm.StateKey{
		DID:   ca.WDID(),
		Nonce: im.Thread().ID,
	}

	state := try.To1(psm.GetPSM(key))
	assert.That(state != nil, "continue issue credential, task not found")

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

func fillIssueCredentialStatus(workerDID string, taskID string, ps *pb.ProtocolStatus) *pb.ProtocolStatus {
	defer err2.Catch(err2.Err(func(err error) {
		glog.Error("Failed to fill issue credential status: ", err)
	}))

	assert.That(ps != nil)

	status := ps

	key := psm.StateKey{
		DID:   workerDID,
		Nonce: taskID,
	}
	credRep := try.To1(data.GetIssueCredRep(key))

	// TODO: save schema id parsed to db? copied from original implementation
	var credOfferMap map[string]interface{}
	dto.FromJSONStr(credRep.CredOffer, &credOfferMap)

	schemaID := credOfferMap["schema_id"].(string)

	attrs := make([]*pb.Protocol_IssuingAttributes_Attribute,
		0, len(credRep.Attributes))
	for _, credAttr := range credRep.Attributes {
		a := &pb.Protocol_IssuingAttributes_Attribute{
			Name:  credAttr.Name,
			Value: credAttr.Value,
		}
		attrs = append(attrs, a)
	}

	status.Status = &pb.ProtocolStatus_IssueCredential{
		IssueCredential: &pb.ProtocolStatus_IssueCredentialStatus{
			CredDefID: credRep.CredDefID,
			SchemaID:  schemaID,
			Attributes: &pb.Protocol_IssuingAttributes{
				Attributes: attrs,
			},
		},
	}

	return status
}
