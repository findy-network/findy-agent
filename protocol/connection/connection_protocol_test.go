package connection

import (
	"encoding/json"
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
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/method"
	model "github.com/findy-network/findy-agent/std/didexchange"
	v1 "github.com/findy-network/findy-common-go/grpc/agency/v1"
	didexchange "github.com/findy-network/findy-common-go/std/didexchange/invitation"
	gomock "github.com/golang/mock/gomock"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type ReceiverMock interface {
	ssi.Agent
	comm.Receiver
}

// go get github.com/golang/mock/mockgen
// go install github.com/golang/mock/mockgen
// /go/bin/mockgen -package connection -source ./protocol/connection/connection_protocol_test.go ReceiverMock > ./protocol/connection/mock_test.go
var res []byte

func sendAndWaitHTTPRequest(urlStr string, msg io.Reader, timeout time.Duration) (data []byte, err error) {
	res, _ = io.ReadAll(msg)
	return []byte{}, nil
}

var (
	ourAgent   = new(ssi.DIDAgent)
	theirAgent = new(ssi.DIDAgent)
	ourDID     core.DID
	theirDID   core.DID

	endpoint = &endp.Addr{
		BasePath: "hostname",
		Service:  "serviceName",
		PlRcvr:   "caDID",
		MsgRcvr:  "caDID",
		ConnID:   "connID",
		VerKey:   "vk",
	}

	re             = regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	requestPayload = re.ReplaceAllString(`{
		"@type":"did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/request",
		"@id":"d3dbb3af-63d4-4c88-85a4-36f0a0b889e0",
		"connection":{
			"DID":"Th7MpTaRZVRYnPiabds81Y",
			"DIDDoc":{
				"@context":"https://w3id.org/did/v1",
				"id":"did:sov:Th7MpTaRZVRYnPiabds81Y",
				"publicKey":[{
					"id":"did:sov:Th7MpTaRZVRYnPiabds81Y#1",
					"type":"Ed25519VerificationKey2018",
					"controller":"did:sov:Th7MpTaRZVRYnPiabds81Y",
					"publicKeyBase58":"FYmoFw55GeQH7SRFa37dkx1d2dZ3zUF8ckg7wmL7ofN4"
				}],
				"service":[{
					"id":"did:sov:Th7MpTaRZVRYnPiabds81Y",
					"type":"IndyAgent",
					"recipientKeys":["FYmoFw55GeQH7SRFa37dkx1d2dZ3zUF8ckg7wmL7ofN4"],
					"serviceEndpoint":"http://example.com"}],
					"authentication":[{
						"type":"Ed25519SignatureAuthentication2018",
						"publicKey":"did:sov:Th7MpTaRZVRYnPiabds81Y#1"
					}]
				}
			},
		"~thread":{
			"thid":"d3dbb3af-63d4-4c88-85a4-36f0a0b889e0"
		}
	}`, "")
	responsePayload = re.ReplaceAllString(`{
		"@type":"did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/response",
		"@id":"659028fa-e988-4941-8361-3aec70b63818",
		"connection~sig":{
			"@type":"did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/signature/1.0/ed25519Sha512_single",
			"signature":"00P0ohRo3XirmZLCGBdKTHIrYxEGuEwfk5VZ_DolaJqDiW9Qo2LzfThwgLyS_9XKC5-XbyQYmEAyxn-hLS3vDw==",
			"sig_data":"AAAAAGNHs3F7IkRJRCI6IkViUDRhWU5lVEhMNnEzODVHdVZwUlYiLCJESUREb2MiOnsiQGNvbnRleHQiOiJodHRwczovL3czaWQub3JnL2RpZC92MSIsImlkIjoiZGlkOnNvdjpFYlA0YVlOZVRITDZxMzg1R3VWcFJWIiwicHVibGljS2V5IjpbeyJpZCI6ImRpZDpzb3Y6RWJQNGFZTmVUSEw2cTM4NUd1VnBSViMxIiwidHlwZSI6IkVkMjU1MTlWZXJpZmljYXRpb25LZXkyMDE4IiwiY29udHJvbGxlciI6ImRpZDpzb3Y6RWJQNGFZTmVUSEw2cTM4NUd1VnBSViIsInB1YmxpY0tleUJhc2U1OCI6IjhRaEZ4S3h5YUZzSnk0Q3l4ZVlYMzRkRkg4b1dxeUJ2MVA0SExRQ3NvZUx5In1dLCJzZXJ2aWNlIjpbeyJpZCI6ImRpZDpzb3Y6RWJQNGFZTmVUSEw2cTM4NUd1VnBSViIsInR5cGUiOiJJbmR5QWdlbnQiLCJyZWNpcGllbnRLZXlzIjpbIjhRaEZ4S3h5YUZzSnk0Q3l4ZVlYMzRkRkg4b1dxeUJ2MVA0SExRQ3NvZUx5Il0sInNlcnZpY2VFbmRwb2ludCI6Imhvc3RuYW1lL3NlcnZpY2VOYW1lL2NhRElEL2NhRElEL2Nvbm5JRCJ9XSwiYXV0aGVudGljYXRpb24iOlt7InR5cGUiOiJFZDI1NTE5U2lnbmF0dXJlQXV0aGVudGljYXRpb24yMDE4IiwicHVibGljS2V5IjoiZGlkOnNvdjpFYlA0YVlOZVRITDZxMzg1R3VWcFJWIzEifV19fQ==",
			"signer":"8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy"},
			"~thread":{
				"thid":"d3dbb3af-63d4-4c88-85a4-36f0a0b889e0"
			}
		}`, "")
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	try.To(psm.Open("data.bolt"))
	comm.SendAndWaitReq = sendAndWaitHTTPRequest

	const walletKey = "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE"

	walletID := fmt.Sprintf("connection-test-agent-1%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, walletKey)
	aw.Create()

	ourAgent.OpenWallet(*aw)
	ourDID = try.To1(ourAgent.NewDID(method.TypeSov, "000000000000000000000000Steward1"))
	ourDID.SetAEndp(service.Addr{Endp: "http://example.com", Key: ourDID.VerKey()})

	walletID = fmt.Sprintf("connection-test-agent-2%d", time.Now().Unix())
	aw = ssi.NewRawWalletCfg(walletID, walletKey)
	aw.Create()

	theirAgent.OpenWallet(*aw)
	theirDID = try.To1(theirAgent.NewDID(method.TypeSov, "000000000000000000000000Steward2"))
}

func tearDown() {
	psm.Close()
	ourAgent.WalletH.Close()
	ourAgent.StorageH.Close()
}

func createInvitation(did core.DID) string {
	return try.To1(didexchange.Build(didexchange.Invitation{
		ID:              "d3dbb3af-63d4-4c88-85a4-36f0a0b889e0",
		Type:            pltype.AriesConnectionInvitation,
		ServiceEndpoint: "http://example.com",
		RecipientKeys:   []string{did.VerKey()},
		Label:           "test",
	}))
}

func TestConnectionRequestor(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invitation := createInvitation(theirDID)
	task, err := createConnectionTask(
		&comm.TaskHeader{TypeID: pltype.CAPairwiseCreate, Method: method.TypeSov},
		&v1.Protocol{
			StartMsg: &v1.Protocol_DIDExchange{
				DIDExchange: &v1.Protocol_DIDExchangeMsg{
					InvitationJSON: invitation,
				},
			},
		},
	)
	assert.INotNil(task)
	assert.INil(err)

	mockReceiver := NewMockReceiverMock(ctrl)
	mockReceiver.EXPECT().CAEndp(task.ID()).Return(endpoint)
	mockReceiver.EXPECT().WDID().Return("Th7MpTaRZVRYnPiabds81Y")
	mockReceiver.EXPECT().WorkerEA().Return(mockReceiver)
	mockReceiver.EXPECT().NewDID(method.TypeSov, "hostname/serviceName/caDID/caDID/connID").Return(ourDID, nil)
	mockReceiver.EXPECT().AddDIDCache(ourDID).Return()
	mockReceiver.EXPECT().NewOutDID(
		"did:sov:", "8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy").Return(
		ourAgent.NewOutDID("did:sov:", "8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy"))
	mockReceiver.EXPECT().AddPipeToPWMap(gomock.Any(), gomock.Any()).Return()

	startConnectionProtocol(mockReceiver, task)

	fmt.Println(string(res), len([]byte(string(res))))
	pipe := sec.Pipe{In: theirDID, Out: ourDID}
	unpacked, _, _ := pipe.Unpack(res)
	fmt.Println(string(unpacked))

	assert.Equal(string(unpacked), requestPayload)
	var request model.RequestImpl
	err = json.Unmarshal(unpacked, &request)
	assert.INil(err)
	assert.Equal("did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/request", request.Type())
	res = []byte{}

	// handle response
	payload := aries.PayloadCreator.NewFromData([]byte(responsePayload))
	mockReceiver.EXPECT().MyDID().Return(ourDID)
	mockReceiver.EXPECT().LoadDID("Th7MpTaRZVRYnPiabds81Y").Return(ourDID)
	mockReceiver.EXPECT().ManagedWallet().AnyTimes().Return(ourAgent.WalletH, ourAgent.StorageH)
	mockReceiver.EXPECT().AddToPWMap(ourDID, gomock.Any(), "d3dbb3af-63d4-4c88-85a4-36f0a0b889e0").Return(pipe)

	err = handleConnectionResponse(comm.Packet{
		Payload:  payload,
		Receiver: mockReceiver,
		Address:  endpoint,
	})

	assert.Equal(len(res), 0)
	assert.INil(err)
}

func TestConnectionInvitor(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReceiver := NewMockReceiverMock(ctrl)

	payload := aries.PayloadCreator.NewFromData([]byte(requestPayload))

	packet := comm.Packet{
		Payload:  payload,
		Receiver: mockReceiver,
		Address:  endpoint,
	}
	outDID := try.To1(theirAgent.NewOutDID("did:sov:Th7MpTaRZVRYnPiabds81Y", "FYmoFw55GeQH7SRFa37dkx1d2dZ3zUF8ckg7wmL7ofN4"))
	pipe := sec.Pipe{In: outDID, Out: theirDID}
	mockReceiver.EXPECT().MyDID().Return(theirDID)
	mockReceiver.EXPECT().FindPWByID("connID").Return(&storage.Connection{}, nil)
	mockReceiver.EXPECT().LoadDID("").Return(theirDID)
	mockReceiver.EXPECT().NewOutDID("did:sov:Th7MpTaRZVRYnPiabds81Y", "FYmoFw55GeQH7SRFa37dkx1d2dZ3zUF8ckg7wmL7ofN4").Return(outDID, nil)
	mockReceiver.EXPECT().AddDIDCache(outDID).Return()
	mockReceiver.EXPECT().ManagedWallet().AnyTimes().Return(ourAgent.WalletH, ourAgent.StorageH)
	mockReceiver.EXPECT().AddToPWMap(theirDID, outDID, "connID").Return(pipe)

	// handler request
	err := handleConnectionRequest(packet)
	assert.INil(err)

	pipe = sec.Pipe{In: ourDID, Out: theirDID}
	fmt.Println(string(res), len([]byte(string(res))))
	unpacked, _, _ := pipe.Unpack(res)
	fmt.Println("\n", string(unpacked))

	//require.Equal(t, string(unpacked), responsePayload)

	signature := &model.ConnectionSignature{
		Type:       "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/signature/1.0/ed25519Sha512_single",
		SignVerKey: "8QhFxKxyaFsJy4CyxeYX34dFH8oWqyBv1P4HLQCsoeLy",
	}
	var response model.ResponseImpl
	try.To(json.Unmarshal(unpacked, &response))
	fmt.Println(response)
	assert.Equal("did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/response", response.Type())
	assert.Equal("d3dbb3af-63d4-4c88-85a4-36f0a0b889e0", response.Thread().ID)
	assert.Equal(signature.Type, response.ConnectionSignature.Type)
	assert.Equal(signature.SignVerKey, response.ConnectionSignature.SignVerKey)
	assert.NotEmpty(response.ConnectionSignature.Signature)
	assert.NotEmpty(response.ConnectionSignature.SignedData)
	assert.NotEmpty(response.ID())

}
