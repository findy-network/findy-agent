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
	ForInvitation(task *taskDIDExchange, caller core.DID) (plToSend didcomm.Payload, plToWait didcomm.Payload)
	ForRequest(taskID string, pw *pairwise.Callee, pipe sec.Pipe) (plToSend didcomm.Payload, plToWait didcomm.Payload)
	ForResponse() (plToSend didcomm.Payload, plToWait didcomm.Payload)
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

func (p *plCreatorDIDExchangeV0) ForInvitation(task *taskDIDExchange, caller core.DID) (plToSend didcomm.Payload, plToWait didcomm.Payload) {
	if task.Role() == pb.Protocol_ADDRESSEE {
		glog.V(3).Infof("it's us who waits connection (%v) to invitation", task.Invitation.ID)
		return nil, createPayload(task.ID(), pltype.AriesConnectionRequest)
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

	return plToSend, createPayload(task.ID(), pltype.AriesConnectionResponse)
}

func (p *plCreatorDIDExchangeV0) ForRequest(taskID string, pw *pairwise.Callee, pipe sec.Pipe) (plToSend didcomm.Payload, plToWait didcomm.Payload) {
	responseMsg := didexchange0.ResponseCreator.Create(didcomm.MsgInit{
		DIDObj:   pw.Callee,
		Nonce:    pw.Name,
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

	return plToSend, plToWait
}

func (p *plCreatorDIDExchangeV0) ForResponse() (plToSend didcomm.Payload, plToWait didcomm.Payload) {
	emptyMsg := aries.MsgCreator.Create(didcomm.MsgInit{})
	plToWait = aries.PayloadCreator.NewMsg(utils.UUID(), pltype.AriesConnectionResponse, emptyMsg)

	return nil, plToWait
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
	return nil, nil
}

/*
   {
       "@type": "https://didcomm.org/didexchange/1.0/request",
       "@id": "b0a169c2-a94a-448b-82e5-aa40518ad656",
       "~thread": {
           "thid": "b0a169c2-a94a-448b-82e5-aa40518ad656",
           "pthid": "c263816b-ddb7-43ac-a2cc-6dcb7cd26e7a"
       },
       "did": "MT2LEeCuvuB3edrAbAY8tR",
       "did_doc~attach": {
           "@id": "63e68517-3409-45d7-9a3e-347482f8f443",
           "mime-type": "application/json",
           "data": {
               "base64": "eyJAY29udGV4dCI6ICJodHRwczovL3czaWQub3JnL2RpZC92MSIsICJpZCI6ICJkaWQ6c292Ok1UMkxFZUN1dnVCM2VkckFiQVk4dFIiLCAicHVibGljS2V5IjogW3siaWQiOiAiZGlkOnNvdjpNVDJMRWVDdXZ1QjNlZHJBYkFZOHRSIzEiLCAidHlwZSI6ICJFZDI1NTE5VmVyaWZpY2F0aW9uS2V5MjAxOCIsICJjb250cm9sbGVyIjogImRpZDpzb3Y6TVQyTEVlQ3V2dUIzZWRyQWJBWTh0UiIsICJwdWJsaWNLZXlCYXNlNTgiOiAiQzlSWFBGRnN3aXhmeGtzYUhnUWJMNVFYd0tXTG9rekRHWVJVaWpDZlNQdkEifV0sICJhdXRoZW50aWNhdGlvbiI6IFt7InR5cGUiOiAiRWQyNTUxOVNpZ25hdHVyZUF1dGhlbnRpY2F0aW9uMjAxOCIsICJwdWJsaWNLZXkiOiAiZGlkOnNvdjpNVDJMRWVDdXZ1QjNlZHJBYkFZOHRSIzEifV0sICJzZXJ2aWNlIjogW3siaWQiOiAiZGlkOnNvdjpNVDJMRWVDdXZ1QjNlZHJBYkFZOHRSO2luZHkiLCAidHlwZSI6ICJJbmR5QWdlbnQiLCAicHJpb3JpdHkiOiAwLCAicmVjaXBpZW50S2V5cyI6IFsiQzlSWFBGRnN3aXhmeGtzYUhnUWJMNVFYd0tXTG9rekRHWVJVaWpDZlNQdkEiXSwgInNlcnZpY2VFbmRwb2ludCI6ICJodHRwOi8vaG9zdC5kb2NrZXIuaW50ZXJuYWw6ODAzMCJ9XX0=",
               "jws": {
                   "header": {
                       "kid": "did:key:z6MkqbgZyVWKHGT95FiGyFNSBAxXktnCDeEZxZLQZ1AgMchY"
                   },
                   "protected": "eyJhbGciOiAiRWREU0EiLCAia2lkIjogImRpZDprZXk6ejZNa3FiZ1p5VldLSEdUOTVGaUd5Rk5TQkF4WGt0bkNEZUVaeFpMUVoxQWdNY2hZIiwgImp3ayI6IHsia3R5IjogIk9LUCIsICJjcnYiOiAiRWQyNTUxOSIsICJ4IjogInBaanVqemtyalN4QVNvRFFkS3ZnQ3JJVlZyVmlwRnRHUGVuOWFYUVM3VXMiLCAia2lkIjogImRpZDprZXk6ejZNa3FiZ1p5VldLSEdUOTVGaUd5Rk5TQkF4WGt0bkNEZUVaeFpMUVoxQWdNY2hZIn19",
                   "signature": "kZm_fAk-DsNvm5K0pXsUSJ2l0Tk0UFrvmvV_Imy0AJsnr3xtaG-13g8dde-av29kMi1GxYTOooK5Uigjd6vkBg"
               }
           }
       },
       "label": "alice.agent"
   },
*/
func (p *plCreatorDIDExchangeV1) ForInvitation(task *taskDIDExchange, caller core.DID) (plToSend didcomm.Payload, plToWait didcomm.Payload) {
	if task.Role() == pb.Protocol_ADDRESSEE {
		glog.V(3).Infof("it's us who waits connection (%v) to invitation", task.Invitation.ID)
		return nil, createPayload(task.ID(), pltype.DIDOrgAriesDIDExchangeRequest)
	}

	// build a connection request message to send to another agent
	msg := didexchange1.NewRequest(&didexchange1.Request{
		Label:  task.Label,
		DID:    caller.Did(),
		DIDDoc: caller.DOC(),
		Thread: &decorator.Thread{ID: task.Invitation.ID, PID: task.Invitation.ID},
	})

	// Create payload to send
	plToSend = aries.PayloadCreator.NewMsg(task.ID(), pltype.DIDOrgAriesDIDExchangeRequest, msg)

	return plToSend, createPayload(task.ID(), pltype.DIDOrgAriesDIDExchangeResponse)
}

func (p *plCreatorDIDExchangeV1) ForRequest(taskID string, pw *pairwise.Callee, pipe sec.Pipe) (plToSend didcomm.Payload, plToWait didcomm.Payload) {
	return nil, nil
}

func (p *plCreatorDIDExchangeV1) ForResponse() (plToSend didcomm.Payload, plToWait didcomm.Payload) {
	return nil, nil
}

func createPayload(id, typeStr string) didcomm.Payload {
	return aries.PayloadCreator.New(
		didcomm.PayloadInit{
			ID:   id,
			Type: typeStr,
		})
}
