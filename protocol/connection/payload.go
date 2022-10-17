package connection

import (
	"encoding/json"
	"errors"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pairwise"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-agent/std/didexchange/signature"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-common-go/std/didexchange/invitation"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type request struct {
	Label    string
	Services []invitation.ServiceEndpoint
	DID      string
	DIDDoc   []byte
}

type response struct {
	DID      string
	Services []invitation.ServiceEndpoint
}

func createPayload(id, typeStr string) didcomm.Payload {
	return aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   id,
			Type: typeStr,
		})
}

func payloadForInvitation(task *taskDIDExchange, caller core.DID) (
	plToSend didcomm.Payload,
	plToWait didcomm.Payload,
) {

	if task.Role() == pb.Protocol_ADDRESSEE {
		glog.V(3).Infof("it's us who waits connection (%v) to invitation", task.Invitation.ID)
		return nil, createPayload(task.ID(), pltype.AriesConnectionRequest)
	}

	// build a connection request message to send to another agent
	msg := didexchange.NewRequest(&didexchange.Request{
		Label: task.Label,
		Connection: &didexchange.Connection{
			DID:    caller.Did(),
			DIDDoc: caller.DOC(),
		},
		// when out-of-bound and did-exchange protocols are supported we
		// should start to save connection_id to Thread.PID
		Thread: &decorator.Thread{ID: task.Invitation.ID},
	})

	// Create payload to send
	plToSend = aries.PayloadCreator.NewMsg(task.ID(), pltype.AriesConnectionRequest, msg)

	return plToSend, createPayload(task.ID(), pltype.AriesConnectionResponse)

}

func requestFromIncoming(ipl didcomm.Payload) (r *request, err error) {
	defer err2.Returnf(&err, "requestFromIncoming")
	switch ipl.Type() {
	case pltype.AriesConnectionRequest:
		{
			req := ipl.MsgHdr().FieldObj().(*didexchange.Request)
			assert.That(len(common.Services(req.Connection.DIDDoc)) > 0)

			doc := req.Connection.DIDDoc
			docBytes := try.To1(json.Marshal(doc))
			return &request{
				DID:    req.Connection.DID,
				DIDDoc: docBytes,
				Label:  req.Label,
				Services: []invitation.ServiceEndpoint{{
					ServiceEndpoint: common.Service(req.Connection.DIDDoc, 0).ServiceEndpoint,
					RecipientKeys:   common.RecipientKeys(req.Connection.DIDDoc, 0),
					RoutingKeys:     common.RoutingKeys(req.Connection.DIDDoc, 0),
				}},
			}, nil

		}

	}
	return nil, nil
}

func payloadForRequest(taskID string, pw *pairwise.Callee, pipe sec.Pipe) (
	plToSend didcomm.Payload,
	plToWait didcomm.Payload,
) {
	responseMsg := didexchange.ResponseCreator.Create(didcomm.MsgInit{
		DIDObj:   pw.Callee,
		Nonce:    pw.Name,
		Name:     pw.Name,
		Endpoint: pw.Endp,
	}).(didcomm.PwMsg)

	res := responseMsg.FieldObj().(*didexchange.Response)
	try.To(signature.Sign(res, pipe)) // we must sign the Response before send it

	// build the response payload, update PSM, and send the PL with sec.Pipe
	plToSend = aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, responseMsg)
	// update the PSM, we are ready at this end for this protocol
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	plToWait = aries.PayloadCreator.NewMsg(taskID, pltype.AriesConnectionResponse, emptyMsg)

	return plToSend, plToWait
}

func verifyResponseFromIncoming(ipl didcomm.Payload) (r *response, err error) {
	resp := ipl.MsgHdr().FieldObj().(*didexchange.Response)
	if !try.To1(signature.Verify(resp)) {
		glog.Error("cannot verify Connection Response signature --> send NACK")
		return nil, errors.New("cannot verify connection response signature")
	}
	assert.That(len(common.Services(resp.Connection.DIDDoc)) > 0)
	// rawDID := strings.TrimPrefix(resp.Connection.DID, "did:sov:")
	// if rawDID == resp.Connection.DID {
	// 	resp.Connection.DID = "did:sov:" + rawDID
	// 	glog.V(3).Infoln("+++ normalizing Did()", rawDID, " ==>", resp.Connection.DID)
	// }

	return &response{
		DID: resp.Connection.DID,
		Services: []invitation.ServiceEndpoint{
			{
				ServiceEndpoint: common.Service(resp.Connection.DIDDoc, 0).ServiceEndpoint,
				RecipientKeys:   common.RecipientKeys(resp.Connection.DIDDoc, 0),
				RoutingKeys:     common.RoutingKeys(resp.Connection.DIDDoc, 0),
			},
		},
	}, nil
}

func payloadForResponse() (
	plToSend didcomm.Payload,
	plToWait didcomm.Payload,
) {
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	plToWait = aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, emptyMsg)

	return nil, plToWait
}
