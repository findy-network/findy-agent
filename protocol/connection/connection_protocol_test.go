package connection

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	storage "github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/method"
	v1 "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/findy-network/findy-common-go/std/didexchange/invitation"
	gomock "github.com/golang/mock/gomock"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type ReceiverMock interface {
	ssi.Agent
	comm.Receiver
}

// TODO: integrate to build pipeline
// how to install and use mockgen:
// go get github.com/golang/mock/mockgen
// go install github.com/golang/mock/mockgen
// /go/bin/mockgen -package connection -source ./protocol/connection/connection_protocol_test.go ReceiverMock > ./protocol/connection/mock_test.go

func sendAndWaitHTTPRequest(urlStr string, msg io.Reader, timeout time.Duration) (data []byte, err error) {
	httpPayload, _ = io.ReadAll(msg)
	return []byte{}, nil
}

var (
	httpPayload []byte

	agents = make([]*ssi.DIDAgent, 0)

	endpoint = &endp.Addr{
		BasePath: "hostname",
		Service:  "serviceName",
		PlRcvr:   "caDID",
		MsgRcvr:  "caDID",
		ConnID:   endpointConnID,
		VerKey:   "vk",
	}
	endpointStr    = "hostname/serviceName/caDID/caDID/" + endpointConnID
	endpointConnID = "d3dbb3af-63d4-4c88-85a4-36f0a0b889e0"

	re = regexp.MustCompile(`[\s\p{Zs}]{1,}`)
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	try.To(psm.Open("MEMORY_data.bolt"))
	comm.SendAndWaitReq = sendAndWaitHTTPRequest
}

func tearDown() {
	psm.Close()
	for _, a := range agents {
		a.WalletH.Close()
		a.StorageH.Close()
	}
}

func createAgent(id string) *ssi.DIDAgent {
	a := new(ssi.DIDAgent)
	const walletKey = "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE"

	walletID := fmt.Sprintf("connection-test-agent-%s-%d", id, time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, walletKey)
	aw.Create()

	a.OpenWallet(*aw)

	agents = append(agents, a)
	return a
}

func readJSONFromFile(filename string) []byte {
	strJSON := string(try.To1(os.ReadFile(filename)))
	return []byte(re.ReplaceAllString(strJSON, ""))
}

// Simulates requestor role
func TestConnectionRequestor(t *testing.T) {
	tests := []struct {
		name                string
		invitationPayload   []byte
		requestPayload      []byte
		requestPayloadType  string
		responsePayload     []byte
		ourSeed             string
		ourDIDStr           string
		theirSeed           string
		theirVerKey         string
		didMethod           method.Type
		invitationID        string
		completePayloadType string
	}{
		{
			name:               "findy-agent v0",
			invitationPayload:  readJSONFromFile("./test_data/v0/invitation-findy.json"),
			requestPayload:     readJSONFromFile("./test_data/v0/request-findy.json"),
			requestPayloadType: pltype.AriesConnectionRequest,
			responsePayload:    readJSONFromFile("./test_data/v0/response-findy.json"),
			ourSeed:            "000000000000000000000000Steward1",
			ourDIDStr:          "Th7MpTaRZVRYnPiabds81Y",
			theirSeed:          "000000000000000000000000Steward2",
			theirVerKey:        "8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy",
			didMethod:          method.TypeSov,
			invitationID:       "d3dbb3af-63d4-4c88-85a4-36f0a0b889e0",
		},
		{
			name:                "findy-agent v1",
			invitationPayload:   readJSONFromFile("./test_data/v1/invitation-findy.json"),
			requestPayload:      readJSONFromFile("./test_data/v1/request-findy.json"),
			requestPayloadType:  pltype.DIDOrgAriesDIDExchangeRequest,
			responsePayload:     readJSONFromFile("./test_data/v1/response-findy.json"),
			ourSeed:             "000000000000000000000000Steward1",
			ourDIDStr:           "Th7MpTaRZVRYnPiabds81Y",
			theirSeed:           "000000000000000000000000Steward2",
			theirVerKey:         "8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy",
			didMethod:           method.TypeSov,
			invitationID:        "d3dbb3af-63d4-4c88-85a4-36f0a0b889e0",
			completePayloadType: pltype.DIDOrgAriesDIDExchangeComplete,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ourAgent := createAgent("our-req" + tt.name)
			theirAgent := createAgent("their-req" + tt.name)

			ourDID := try.To1(ourAgent.NewDID(tt.didMethod, tt.ourSeed))
			ourDID.SetAEndp(service.Addr{Endp: "http://example.com", Key: ourDID.VerKey()})
			theirDID := try.To1(theirAgent.NewDID(tt.didMethod, tt.theirSeed))

			// 1. create invitation for "them" and create task
			inv := try.To1(invitation.Translate(string(tt.invitationPayload)))
			task, err := createConnectionTask(
				&comm.TaskHeader{TypeID: pltype.CAPairwiseCreate, Method: tt.didMethod},
				&v1.Protocol{
					StartMsg: &v1.Protocol_DIDExchange{
						DIDExchange: &v1.Protocol_DIDExchangeMsg{
							InvitationJSON: try.To1(invitation.Build(inv)),
						},
					},
				},
			)
			assert.INotNil(task)
			assert.NoError(err)

			// 2. Start protocol -> expect that request message is sent to other end
			mockReceiver := NewMockReceiverMock(ctrl)
			mockReceiver.EXPECT().CAEndp(task.ID()).Return(endpoint)
			mockReceiver.EXPECT().WDID().Return(tt.ourDIDStr)
			mockReceiver.EXPECT().WorkerEA().Return(mockReceiver)
			mockReceiver.EXPECT().NewDID(tt.didMethod, endpointStr).Return(ourDID, nil)
			mockReceiver.EXPECT().AddDIDCache(ourDID).Return()
			mockReceiver.EXPECT().NewOutDID(tt.didMethod.DIDString(), tt.theirVerKey).Return(
				ourAgent.NewOutDID(tt.didMethod.DIDString(), tt.theirVerKey))
			mockReceiver.EXPECT().AddPipeToPWMap(gomock.Any(), gomock.Any()).Return()

			startConnectionProtocol(mockReceiver, task)

			pipe := sec.Pipe{In: theirDID, Out: ourDID}
			unpacked, _, _ := pipe.Unpack(httpPayload)
			// TODO: compare contents
			// assert.Equal(string(unpacked), string(tt.requestPayload))

			request := aries.PayloadCreator.NewFromData(unpacked)
			assert.NoError(err)
			assert.Equal(request.Type(), tt.requestPayloadType)
			httpPayload = []byte{}

			// 3. Handle response -> expect that no message is sent to other end
			payload := aries.PayloadCreator.NewFromData(tt.responsePayload)
			mockReceiver.EXPECT().MyDID().Return(ourDID)
			mockReceiver.EXPECT().LoadDID(tt.ourDIDStr).Return(ourDID)
			mockReceiver.EXPECT().ManagedWallet().AnyTimes().Return(ourAgent.WalletH, ourAgent.StorageH)
			mockReceiver.EXPECT().AddToPWMap(ourDID, gomock.Any(), tt.invitationID).Return(pipe)

			err = handleConnectionResponse(comm.Packet{
				Payload:  payload,
				Receiver: mockReceiver,
				Address:  endpoint,
			})
			assert.NoError(err)
			if tt.completePayloadType == "" {
				assert.Equal(len(httpPayload), 0)
			} else {
				unpacked, _, _ := pipe.Unpack(httpPayload)

				complete := aries.PayloadCreator.NewFromData(unpacked)
				assert.Equal(complete.Type(), tt.completePayloadType)
			}
		})
	}
}

// Simulates invitor role
func TestConnectionInvitor(t *testing.T) {
	tests := []struct {
		name                string
		requestPayload      []byte
		responsePayload     []byte
		responsePayloadType string
		ourSeed             string
		ourDIDStr           string
		theirSeed           string
		theirVerKey         string
		didMethod           method.Type
		invitationID        string
	}{
		{
			name:                "findy-agent v0",
			requestPayload:      readJSONFromFile("./test_data/v0/request-findy.json"),
			responsePayload:     readJSONFromFile("./test_data/v0/response-findy.json"),
			responsePayloadType: pltype.AriesConnectionResponse,
			ourSeed:             "000000000000000000000000Steward1",
			ourDIDStr:           "Th7MpTaRZVRYnPiabds81Y",
			theirSeed:           "000000000000000000000000Steward2",
			theirVerKey:         "8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy",
			didMethod:           method.TypeSov,
			invitationID:        "d3dbb3af-63d4-4c88-85a4-36f0a0b889e0",
		},
		{
			name:                "findy-agent v1",
			requestPayload:      readJSONFromFile("./test_data/v1/request-findy.json"),
			responsePayload:     readJSONFromFile("./test_data/v1/response-findy.json"),
			responsePayloadType: pltype.DIDOrgAriesDIDExchangeResponse,
			ourSeed:             "000000000000000000000000Steward1",
			ourDIDStr:           "Th7MpTaRZVRYnPiabds81Y",
			theirSeed:           "000000000000000000000000Steward2",
			theirVerKey:         "8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy",
			didMethod:           method.TypeSov,
			invitationID:        "d3dbb3af-63d4-4c88-85a4-36f0a0b889e0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ourAgent := createAgent("our-inv" + tt.name)
			theirAgent := createAgent("their-inv" + tt.name)

			// TODO: add test step for invitation creation
			ourDID := try.To1(ourAgent.NewDID(tt.didMethod, tt.ourSeed))
			ourDID.SetAEndp(service.Addr{Endp: "http://example.com", Key: ourDID.VerKey()})
			theirDID := try.To1(theirAgent.NewDID(tt.didMethod, tt.theirSeed))
			mockReceiver := NewMockReceiverMock(ctrl)

			// Handle request -> expect that response is sent to other end
			payload := aries.PayloadCreator.NewFromData(tt.requestPayload)

			packet := comm.Packet{
				Payload:  payload,
				Receiver: mockReceiver,
				Address:  endpoint,
			}
			outDID := try.To1(theirAgent.NewOutDID(ourDID.String(), ourDID.VerKey()))
			mockReceiver.EXPECT().MyDID().Return(theirDID)
			mockReceiver.EXPECT().FindPWByID(endpointConnID).Return(&storage.Connection{
				MyDID: theirDID.String(),
			}, nil)
			mockReceiver.EXPECT().LoadDID(theirDID.String()).Return(theirDID)
			mockReceiver.EXPECT().NewOutDID(ourDID.String(), ourDID.VerKey()).Return(outDID, nil)
			mockReceiver.EXPECT().AddDIDCache(outDID).Return()
			mockReceiver.EXPECT().ManagedWallet().AnyTimes().Return(theirAgent.WalletH, theirAgent.StorageH)
			mockReceiver.EXPECT().AddToPWMap(theirDID, outDID, endpointConnID).Return(sec.Pipe{In: outDID, Out: theirDID})

			err := handleConnectionRequest(packet)
			assert.NoError(err)

			pipe := sec.Pipe{In: ourDID, Out: theirDID}
			unpacked, _, _ := pipe.Unpack(httpPayload)
			httpPayload = []byte{}

			request := aries.PayloadCreator.NewFromData(unpacked)
			assert.NoError(err)
			assert.Equal(request.Type(), tt.responsePayloadType)

			// TODO
			// signature := &model.ConnectionSignature{
			// 	Type:       "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/signature/1.0/ed25519Sha512_single",
			// 	SignVerKey: tt.theirVerKey,
			// }
			// var response model.ResponseImpl
			// try.To(json.Unmarshal(unpacked, &response))

			// assert.Equal(response.Type(), tt.responsePayloadType)
			// assert.Equal(response.Thread().ID, tt.invitationID)
			// assert.Equal(signature.Type, response.ConnectionSignature.Type)
			// assert.Equal(signature.SignVerKey, response.ConnectionSignature.SignVerKey)
			// assert.NotEmpty(response.ConnectionSignature.Signature)
			// assert.NotEmpty(response.ConnectionSignature.SignedData)
			// assert.NotEmpty(response.ID())
		})
	}

}
