package connection

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pairwise"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	didexchange0 "github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-agent/std/didexchange/signature"
	"github.com/findy-network/findy-agent/std/didexchange1"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-common-go/std/didexchange/invitation"
	"github.com/golang/glog"
	decorator1 "github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/fingerprint"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

type Incoming struct {
	Label         string
	Endpoint      string
	RecipientKeys []string
	RoutingKeys   []string
	DID           string
	DIDDoc        []byte
}

type PayloadCreator interface {
	ParseInvitation(pl invitation.Invitation) (r *Incoming, err error)
	ParseIncoming(pl didcomm.Payload) (r *Incoming, err error)
	ForInvitation(task *taskDIDExchange, caller core.DID) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error)
	ForRequest(taskID string, pw *pairwise.Callee, pipe sec.Pipe) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error)
	ForResponse(taskID string) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error)
}

func PayloadCreatorForMessageType(msgType string) (c PayloadCreator, err error) {
	switch {
	case strings.HasPrefix(msgType, pltype.AriesConnection) ||
		strings.HasPrefix(msgType, pltype.DIDOrgAriesConnection):
		{
			return &plCreatorDIDExchangeV0{}, nil
		}
	case strings.HasPrefix(msgType, pltype.AriesOutOfBand) ||
		strings.HasPrefix(msgType, pltype.DIDOrgAriesOutOfBand) ||
		strings.HasPrefix(msgType, pltype.AriesDIDExchange) ||
		strings.HasPrefix(msgType, pltype.DIDOrgAriesDIDExchange):
		{
			return &plCreatorDIDExchangeV1{}, nil
		}
	}
	return nil, fmt.Errorf("no creator for msg type %s", msgType)
}

type plCreatorDIDExchangeV0 struct{}

func (p *plCreatorDIDExchangeV0) ParseInvitation(pl invitation.Invitation) (r *Incoming, err error) {
	assert.SNotEmpty(pl.ServiceEndpoint())
	return &Incoming{
		Label:         pl.Label(),
		Endpoint:      pl.ServiceEndpoint()[0].ServiceEndpoint,
		RecipientKeys: pl.ServiceEndpoint()[0].RecipientKeys,
		RoutingKeys:   pl.ServiceEndpoint()[0].RoutingKeys,
	}, nil
}

func (p *plCreatorDIDExchangeV0) ParseIncoming(pl didcomm.Payload) (r *Incoming, err error) {
	defer err2.Returnf(&err, "DIDExchangeV0 ParseIncoming")
	switch pl.Type() {
	case pltype.AriesConnectionRequest, pltype.DIDOrgAriesConnectionRequest:
		{
			req := pl.MsgHdr().FieldObj().(*didexchange0.Request)
			assert.That(len(common.Services(req.Connection.DIDDoc)) > 0)

			doc := req.Connection.DIDDoc
			docBytes := try.To1(json.Marshal(doc))
			return &Incoming{
				DID:           req.Connection.DID,
				DIDDoc:        docBytes,
				Label:         req.Label,
				Endpoint:      common.Service(req.Connection.DIDDoc, 0).ServiceEndpoint,
				RecipientKeys: common.RecipientKeys(req.Connection.DIDDoc, 0),
				RoutingKeys:   common.RoutingKeys(req.Connection.DIDDoc, 0),
			}, nil
		}
	case pltype.AriesConnectionResponse, pltype.DIDOrgAriesConnectionResponse:
		{
			resp := pl.MsgHdr().FieldObj().(*didexchange0.Response)
			if !try.To1(signature.Verify(resp)) {
				glog.Error("cannot verify Connection Response signature --> send NACK")
				return nil, errors.New("cannot verify connection response signature")
			}
			assert.That(len(common.Services(resp.Connection.DIDDoc)) > 0)

			doc := resp.Connection.DIDDoc
			docBytes := try.To1(json.Marshal(doc))

			return &Incoming{
				DID:           resp.Connection.DID,
				DIDDoc:        docBytes,
				Endpoint:      common.Service(resp.Connection.DIDDoc, 0).ServiceEndpoint,
				RecipientKeys: common.RecipientKeys(resp.Connection.DIDDoc, 0),
				RoutingKeys:   common.RoutingKeys(resp.Connection.DIDDoc, 0),
			}, nil
		}

	}
	return nil, fmt.Errorf("no match for msg type %s", pl.Type())
}

func (p *plCreatorDIDExchangeV0) ForInvitation(task *taskDIDExchange, caller core.DID) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error) {
	if task.Role() == pb.Protocol_ADDRESSEE {
		glog.V(3).Infof("it's us who waits connection (%v) to invitation", task.Invitation.ID)
		return nil, createPayload(task.ID(), pltype.AriesConnectionRequest), nil
	}

	// build a connection request message to send to another agent
	msg := didexchange0.NewRequest(&didexchange0.Request{
		Label: task.Label,
		Connection: &didexchange0.Connection{
			DID:    caller.Did(),
			DIDDoc: caller.DOC(),
		},
		// when out-of-bound and did-exchange protocols are supported we
		// should start to save connection_id to Thread.PID
		Thread: &decorator.Thread{ID: task.Invitation.ID},
	})

	// Create payload to send
	plToSend = aries.PayloadCreator.NewMsg(task.ID(), pltype.AriesConnectionRequest, msg)

	return plToSend, createPayload(task.ID(), pltype.AriesConnectionResponse), nil
}

func (p *plCreatorDIDExchangeV0) ForRequest(taskID string, pw *pairwise.Callee, pipe sec.Pipe) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error) {
	responseMsg := didexchange0.ResponseCreator.Create(didcomm.MsgInit{
		DIDObj:   pw.Callee,
		Nonce:    taskID,
		Name:     pw.Name,
		Endpoint: pw.Endp,
	}).(didcomm.PwMsg)

	res := responseMsg.FieldObj().(*didexchange0.Response)
	try.To(signature.Sign(res, pipe)) // we must sign the Response before send it

	// build the response payload, update PSM, and send the PL with sec.Pipe
	plToSend = aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, responseMsg)
	// update the PSM, we are ready at this end for this protocol
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	plToWait = aries.PayloadCreator.NewMsg(taskID, pltype.AriesConnectionResponse, emptyMsg)

	return plToSend, plToWait, nil
}

func (p *plCreatorDIDExchangeV0) ForResponse(taskID string) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error) {
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	plToWait = aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, emptyMsg)

	return nil, plToWait, nil
}

type plCreatorDIDExchangeV1 struct{}

func didKeysToB58(keys []string) []string {
	for index, key := range keys {
		assert.That(strings.HasPrefix(key, "did:key"))

		keyBytes := try.To1(fingerprint.PubKeyFromDIDKey(key))
		keys[index] = base58.Encode(keyBytes)
	}
	return keys
}

func (p *plCreatorDIDExchangeV1) ParseInvitation(pl invitation.Invitation) (r *Incoming, err error) {
	assert.SNotEmpty(pl.ServiceEndpoint())
	return &Incoming{
		Label:         pl.Label(),
		Endpoint:      pl.ServiceEndpoint()[0].ServiceEndpoint,
		RecipientKeys: didKeysToB58(pl.ServiceEndpoint()[0].RecipientKeys),
		RoutingKeys:   didKeysToB58(pl.ServiceEndpoint()[0].RoutingKeys),
	}, nil
}

func (p *plCreatorDIDExchangeV1) ParseIncoming(pl didcomm.Payload) (r *Incoming, err error) {
	defer err2.Returnf(&err, "DIDExchangeV0 ParseIncoming")
	switch pl.Type() {
	case pltype.AriesDIDExchangeRequest, pltype.DIDOrgAriesDIDExchangeRequest:
		{
			pwMsg := pl.MsgHdr().(didcomm.PwMsg)
			doc := try.To1(pwMsg.DIDDocument())
			assert.That(len(common.Services(doc)) > 0)
			docBytes := try.To1(json.Marshal(doc))

			req := pl.MsgHdr().FieldObj().(*didexchange1.Request)
			return &Incoming{
				DID:           req.DID,
				DIDDoc:        docBytes,
				Label:         req.Label,
				Endpoint:      common.Service(doc, 0).ServiceEndpoint,
				RecipientKeys: common.RecipientKeys(doc, 0),
				RoutingKeys:   common.RoutingKeys(doc, 0),
			}, nil
		}
	case pltype.AriesDIDExchangeResponse, pltype.DIDOrgAriesDIDExchangeResponse:
		{
			pwMsg := pl.MsgHdr().(didcomm.PwMsg)
			doc := try.To1(pwMsg.DIDDocument())
			assert.That(len(common.Services(doc)) > 0)
			docBytes := try.To1(json.Marshal(doc))

			resp := pl.MsgHdr().FieldObj().(*didexchange1.Response)
			// TODO
			// if !try.To1(signature.Verify(resp)) {
			// 	glog.Error("cannot verify Connection Response signature --> send NACK")
			// 	return nil, errors.New("cannot verify connection response signature")
			// }
			assert.That(len(common.Services(doc)) > 0)

			return &Incoming{
				DID:           resp.DID,
				DIDDoc:        docBytes,
				Endpoint:      common.Service(doc, 0).ServiceEndpoint,
				RecipientKeys: common.RecipientKeys(doc, 0),
				RoutingKeys:   common.RoutingKeys(doc, 0),
			}, nil
		}

	}
	return nil, fmt.Errorf("no match for msg type %s", pl.Type())
}

func (p *plCreatorDIDExchangeV1) ForInvitation(task *taskDIDExchange, caller core.DID) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error) {
	defer err2.Returnf(&err, "V1 pl for invitation")
	if task.Role() == pb.Protocol_ADDRESSEE {
		glog.V(3).Infof("it's us who waits connection (%v) to invitation", task.Invitation.ID)
		return nil, createPayload(task.ID(), pltype.DIDOrgAriesDIDExchangeRequest), nil
	}

	// build a connection request message to send to another agent
	msg := didexchange1.NewRequest(caller.DOC(), &didexchange1.Request{
		Label:  task.Label,
		DID:    caller.Did(),
		Thread: &decorator1.Thread{ID: task.Invitation.ID, PID: task.Invitation.ID},
	})
	res := msg.FieldObj().(*didexchange1.Request)
	try.To(signature.SignRequestV1(res, caller))

	// Create payload to send
	plToSend = aries.PayloadCreator.NewMsg(task.ID(), pltype.DIDOrgAriesDIDExchangeRequest, msg)

	return plToSend, createPayload(task.ID(), pltype.DIDOrgAriesDIDExchangeResponse), nil
}

func (p *plCreatorDIDExchangeV1) ForRequest(taskID string, pw *pairwise.Callee, pipe sec.Pipe) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error) {
	assert.That(pw.Callee.DOC() != nil)
	assert.NotEmpty(pw.Callee.Did())

	msg := didexchange1.NewResponse(pw.Callee.DOC(), &didexchange1.Response{
		DID:    pw.Callee.Did(),
		Thread: &decorator1.Thread{ID: taskID, PID: taskID},
	})

	res := msg.FieldObj().(*didexchange1.Response)
	try.To(signature.SignResponseV1(res, pw.Callee))

	plToSend = aries.PayloadCreator.NewMsg(taskID, pltype.DIDOrgAriesDIDExchangeResponse, msg)

	return plToSend, createPayload(taskID, pltype.DIDOrgAriesDIDExchangeComplete), nil
}

func (p *plCreatorDIDExchangeV1) ForResponse(taskID string) (plToSend didcomm.Payload, plToWait didcomm.Payload, err error) {
	msg := didexchange1.NewComplete(&didexchange1.Complete{
		Thread: &decorator1.Thread{ID: taskID, PID: taskID},
	})
	plToSend = aries.PayloadCreator.NewMsg(taskID, pltype.DIDOrgAriesDIDExchangeComplete, msg)
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	plToWait = aries.PayloadCreator.NewMsg(taskID, pltype.DIDOrgAriesDIDExchangeComplete, emptyMsg)

	return plToSend, plToWait, nil
}

func createPayload(id, typeStr string) didcomm.Payload {
	return aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   id,
			Type: typeStr,
		})
}
