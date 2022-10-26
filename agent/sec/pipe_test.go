package sec_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/method"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	v0 "github.com/findy-network/findy-agent/std/didexchange/v0"
	"github.com/findy-network/findy-common-go/dto"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

const (
	key1 = "did:key:z6Mkj5J66HkkrfSH2Ld63zvBbnEvDSk5E3cfhKRt7213Reso"
	key2 = "did:key:z6MkqQ81wZSsjWeTk4MnPVow3Jyydp31AP7qNj3WvUtrdejx"
	// key3 = "did:key:z6MkmPrHsyXEeujwhpMGSyyxmixpuqUYQ2QPfj3Y3gFPugNp"
	// key4 = "did:key:z6MkuMg4H1GH2XdLPuBMcuDvWx18NNHFie37PN37GP7V1L4G"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	home := utils.IndyBaseDir()
	removeFiles(home, "/.indy_client/wallet/pipe-test-agent*")

	err2.SetTracers(nil)

}

func removeFiles(home, nameFilter string) {
	filter := filepath.Join(home, nameFilter)
	files, _ := filepath.Glob(filter)
	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			panic(err)
		}
	}
}

var (
	agent, agent2 = new(ssi.DIDAgent), new(ssi.DIDAgent)
)

func setUp() {
	err2.SetTracers(os.Stderr)
	assert.D = assert.AsserterCallerInfo
	assert.DefaultAsserter = assert.AsserterFormattedCallerInfo

	// init pipe package, TODO: try to find out how to get media profile
	// from...
	sec.Init(transport.MediaTypeProfileDIDCommAIP1)

	// first, create agent 1 with the storages
	walletID := fmt.Sprintf("pipe-test-agent-1%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()
	agent.OpenWallet(*aw)

	// second, create agent 2 with the storages
	walletID2 := fmt.Sprintf("pipe-test-agent-2%d", time.Now().Unix())
	aw2 := ssi.NewRawWalletCfg(walletID2, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw2.Create()
	agent2.OpenWallet(*aw2)
}

func TestNewPipe(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	tests := []struct {
		name   string
		method method.Type
	}{
		{"key method", method.TypeKey},
		{"peer method", method.TypePeer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			didIn, _ := agent.NewDID(tt.method, "")
			println(didIn.URI())
			didOut, _ := agent.NewDID(tt.method, "")
			println(didOut.URI())
			didRoute1, _ := agent.NewDID(tt.method, "")
			println(didRoute1.URI())
			didRoute2, _ := agent.NewDID(tt.method, "")
			println(didRoute2.URI())

			assert.INotNil(didIn)
			assert.INotNil(didOut)
			assert.INotNil(didRoute1)
			assert.INotNil(didRoute2)

			message := []byte("message")

			p := sec.Pipe{In: didIn, Out: didOut}

			packed, _, err := p.Pack(message)
			assert.NoError(err)

			received, _, err := p.Unpack(packed)
			assert.NoError(err)
			assert.DeepEqual(message, received)
		})
	}
}

func TestResolve(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	vdr := agent.VDR() // .Registry()
	docR := try.To1(vdr.Registry().Resolve(key1))
	assert.INotNil(docR)
	bytes := try.To1(docR.DIDDocument.JSONBytes())
	assert.INotNil(bytes)
}

func TestPackTowardsPubKeyOnly(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	didIn, _ := agent.NewDID(method.TypeKey, "")
	assert.INotNil(didIn)
	println(didIn.String())
	didOut, err := agent.NewOutDID(key2)
	assert.NoError(err)
	assert.INotNil(didOut)
	println(didOut.String())

	message := []byte("message")

	p := sec.Pipe{In: didIn, Out: didOut}

	packed, _ := try.To2(p.Pack(message))
	assert.INotNil(packed)
}

func TestSignVerifyWithSeparatedWallets(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	// we need to use two different agents that we have 2 different key and
	// other storages. The AFGO (Tink) needs to have other agent's PubKey saved
	// to its storage (to have key handle) that it can e.g. verify signatur.
	// The access to public key is not enough. It must first stored.

	// create first agent2's input DID
	didIn2, _ := agent2.NewDID(method.TypeKey, "")
	assert.INotNil(didIn2)
	println("in2: ", didIn2.URI())

	didIn, _ := agent.NewDID(method.TypeKey, "")
	assert.INotNil(didIn)
	println("in: ", didIn.URI())

	// give agent2's prime DID (input) to agent1's out DID
	didOut, err := agent.NewOutDID(didIn2.URI())
	assert.NoError(err)
	assert.INotNil(didOut)
	println("out: ", didOut.URI())

	// similarly, give agent1's in-DID to agent2's out-DID
	didOut2, err := agent2.NewOutDID(didIn.URI())
	assert.NoError(err)
	assert.INotNil(didOut2)
	println("out2: ", didOut2.URI())

	message := []byte("message")

	p := sec.Pipe{In: didIn, Out: didOut}
	p2 := sec.Pipe{In: didIn2, Out: didOut2}

	packed, _ := try.To2(p.Pack(message))
	assert.INotNil(packed)
	received, _ := try.To2(p2.Unpack(packed))
	assert.DeepEqual(message, received)

	sign, _, err := p.Sign(message)
	assert.NoError(err)

	// Signature verification must done from p2 because p2 has only pubKey of
	// the DID in the 'wallet' where p2 is connected to. This way the test
	// follows the real world situation
	ok, _, err := p2.Verify(message, sign)
	assert.NoError(err)

	assert.That(ok)
}

func TestIndyPipe(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	defer err2.CatchAll(func(err error) {
		println(err)
		utils.Settings.SetDIDMethod(method.TypeSov)
	}, func(v any) {
		println(v)
		utils.Settings.SetDIDMethod(method.TypeSov)
	})
	utils.Settings.SetDIDMethod(method.TypeSov)

	didIn, _ := agent.NewDID(method.TypeSov, "")
	str := didIn.String()
	assert.NotEmpty(str)
	println(str)

	didIn2, _ := agent2.NewDID(method.TypeSov, "")
	did2 := didIn2.String()
	assert.NotEmpty(did2)
	println(did2)
	println(didIn2.VerKey())

	did2 = "did:sov:"
	didOut, err := agent.NewOutDID(did2, didIn2.VerKey())
	assert.NoError(err)

	p := sec.Pipe{In: didIn, Out: didOut}

	message := []byte("message")

	packed, _, err := p.Pack(message)
	assert.NoError(err)
	assert.INotNil(packed)

	didOut2, err := agent2.NewOutDID(did2, didIn.VerKey())
	assert.NoError(err)

	p2 := sec.Pipe{In: didIn2, Out: didOut2}
	received, _ := try.To2(p2.Unpack(packed))
	assert.DeepEqual(message, received)

	sign, vk, err := p.Sign(message)
	assert.NoError(err)
	assert.Equal(p2.Out.VerKey(), vk)

	// Signature verification must be done from p2 because p2 has only pubKey
	// of the DID in the 'wallet' where p2 is connected to. This way the test
	// follows the real world situation
	ok, _, err := p2.Verify(message, sign)
	assert.NoError(err)

	assert.That(ok)

	p3 := sec.Pipe{Out: didOut2}

	// Signature verification must be done from p2 because p2 has only pubKey
	// Now we test the pipe which have only one end, no sender
	ok, _, err = p3.Verify(message, sign)
	assert.NoError(err)
	assert.That(ok)
}

type protected struct {
	Recipients []struct {
		Header struct {
			Kid string `json:"kid"`
		} `json:"header"`
	} `json:"recipients"`
}

func getRecipientKeysFromBytes(msg []byte) (keys []string, err error) {
	data := make(map[string]interface{})
	if err = json.Unmarshal(msg, &data); err != nil {
		return nil, err
	}
	return getRecipientKeys(data)
}

func getRecipientKeys(msg map[string]interface{}) (keys []string, err error) {
	defer err2.Return(&err)

	protData := try.To1(utils.DecodeB64(msg["protected"].(string)))

	data := protected{}
	try.To(json.Unmarshal(protData, &data))

	keys = make([]string, 0)
	for _, recipient := range data.Recipients {
		keys = append(keys, recipient.Header.Kid)
	}
	return keys, nil
}

//nolint:funlen
func TestPipe_pack(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	defer err2.CatchAll(func(err error) {
		println(err)
		utils.Settings.SetDIDMethod(method.TypeSov)
	}, func(v any) {
		println(v)
		utils.Settings.SetDIDMethod(method.TypeSov)
	})
	utils.Settings.SetDIDMethod(method.TypePeer)

	// Create test wallet and routing keys
	walletID := fmt.Sprintf("pipe-test-agent-%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()
	a := ssi.DIDAgent{}
	a.OpenWallet(*aw)

	didIn, _ := a.NewDID(method.TypeSov, "")
	didOut, _ := a.NewDID(method.TypeSov, "")
	didRoute1, _ := a.NewDID(method.TypeSov, "")
	didRoute2, _ := a.NewDID(method.TypeSov, "")

	// Packing pipe with two routing keys
	packPipe := sec.NewPipeByVerkey(didIn, didOut.VerKey(),
		[]string{didRoute1.VerKey(), didRoute2.VerKey()})

	// Simulate actual aries message
	plID := utils.UUID()
	doc := didIn.NewDoc(service.Addr{
		Endp: "http://example.com",
		Key:  didIn.VerKey(),
	})
	msg := v0.NewRequest(&v0.Request{
		Label: "test",
		Connection: &v0.Connection{
			DID:    didIn.Did(),
			DIDDoc: doc,
		},
		Thread: &decorator.Thread{ID: utils.UUID()},
	})
	pl := aries.PayloadCreator.NewMsg(plID, pltype.AriesConnectionRequest, msg)

	// Pack message
	route2bytes, _, err := packPipe.Pack(pl.JSON())
	assert.NoError(err)
	assert.That(len(route2bytes) > 0)

	// Unpack forward message with last routing key
	route2Keys, err := getRecipientKeysFromBytes(route2bytes)
	assert.NoError(err)
	assert.That(len(route2Keys) == 1)
	assert.Equal(didRoute2.VerKey(), route2Keys[0])

	route1UnpackPipe := sec.NewPipeByVerkey(didRoute2, didIn.VerKey(), []string{})
	route1FwBytes, _, err := route1UnpackPipe.Unpack(route2bytes)
	assert.NoError(err)

	// Unpack next forward message with first routing key
	route1FwdMsg := aries.PayloadCreator.NewFromData(route1FwBytes).MsgHdr().FieldObj().(*common.Forward)
	route1Bytes := route1FwdMsg.Msg
	route1Keys, err := getRecipientKeys(route1Bytes)
	assert.NoError(err)
	assert.That(len(route1Keys) == 1)
	assert.Equal(didRoute1.VerKey(), route1Keys[0])
	assert.Equal(didRoute1.VerKey(), route1FwdMsg.To)

	dstUnpackPipe := sec.NewPipeByVerkey(didOut, didIn.VerKey(), []string{})
	dstFwBytes, _, err := dstUnpackPipe.Unpack(dto.ToJSONBytes(route1Bytes))
	assert.NoError(err)

	// Unpack final (anon-crypted) forward message with destination key
	dstFwdMsg := aries.PayloadCreator.NewFromData(dstFwBytes).MsgHdr().FieldObj().(*common.Forward)
	dstPackedBytes := dto.ToJSONBytes(dstFwdMsg.Msg)
	dstFwdKeys, err := getRecipientKeys(dstFwdMsg.Msg)
	assert.NoError(err)
	assert.That(len(dstFwdKeys) == 1)
	assert.Equal(didOut.VerKey(), dstFwdKeys[0])
	assert.Equal(didOut.VerKey(), dstFwdMsg.To)

	// Unpack final (auth-crypted) message with destination key
	dstBytes, _, err := dstUnpackPipe.Unpack(dstPackedBytes)
	assert.NoError(err)
	dstKeys, err := getRecipientKeysFromBytes(dstPackedBytes)
	assert.NoError(err)
	assert.That(len(dstFwdKeys) == 1)
	assert.Equal(didOut.VerKey(), dstKeys[0])

	dstMsg := aries.PayloadCreator.NewFromData(dstBytes)
	assert.That(dstMsg.MsgHdr().Type() == pltype.AriesConnectionRequest)
	assert.That(dstMsg.MsgHdr().ID() == plID)
	assert.That(dstMsg.MsgHdr().FieldObj().(*v0.Request).Label == "test")
}

func TestPipe_packPeer(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	defer err2.CatchAll(func(err error) {
		fmt.Println(err)
		t.Fail()
	}, func(v any) {
		fmt.Println(v)
		t.Error(v)
	})
	// Create test wallet and routing keys
	walletID := fmt.Sprintf("pipe-test-agent-%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()
	a := ssi.DIDAgent{}
	a.OpenWallet(*aw)

	wrongAgentDid, _ := agent.NewDID(method.TypePeer, "http://example.com")

	didIn, _ := a.NewDID(method.TypePeer, "http://example.com")
	didOut, _ := a.NewDID(method.TypePeer, "http://example.com")

	didRoute1, _ := a.NewDID(method.TypeKey, "")
	didRoute2, _ := a.NewDID(method.TypeKey, "")

	outDoc := didOut.DOC().(*did.Doc)
	outDoc.Service[0].RoutingKeys = []string{didRoute1.VerKey(), didRoute2.VerKey()}
	docBytes := try.To1(json.Marshal(outDoc))
	out, err := a.NewOutDID(didOut.URI(), string(docBytes))
	assert.NoError(err)

	packPipe := &sec.Pipe{
		In:  didIn,
		Out: out,
	}

	// Simulate actual aries message
	plID := utils.UUID()
	doc, ok := didIn.DOC().(*did.Doc)
	assert.That(ok)

	msg := v0.NewRequest(&v0.Request{
		Label: "test",
		Connection: &v0.Connection{
			DID:    didIn.Did(),
			DIDDoc: doc,
		},
		Thread: &decorator.Thread{ID: utils.UUID()},
	})
	pl := aries.PayloadCreator.NewMsg(plID, pltype.AriesConnectionRequest, msg)

	// Pack message
	route2bytes, _, err := packPipe.Pack(pl.JSON())
	assert.NoError(err)
	assert.That(len(route2bytes) > 0)

	// Unpack forward message with last routing key
	route2Keys, err := getRecipientKeysFromBytes(route2bytes)
	assert.NoError(err)
	assert.SLen(route2Keys, 3)

	// these two rows haven't yet checked. API has changed and these haven't
	// tested yet:
	// lastRouteKey := route2Keys[0]
	// assert.Equal( didRoute2.VerKey(), lastRouteKey)

	firstUnpackPipe := &sec.Pipe{ // this is receiver pipe, i.e. opposite direction
		In:  didRoute2, // start with last route
		Out: didIn,     // and now reiver is original sender
	}

	route1FwBytes, _, err := firstUnpackPipe.Unpack(route2bytes)
	assert.NoError(err)
	assert.INotNil(route1FwBytes)

	secondUnpackPipe := &sec.Pipe{ // this is receiver pipe, i.e. opposite direction
		In:  didRoute1, // start with last route
		Out: didIn,     // and now reiver is original sender
	}

	route2FwBytes, _, err := secondUnpackPipe.Unpack(route2bytes)
	assert.NoError(err)
	assert.INotNil(route2FwBytes)

	wrongUnpackPipe := &sec.Pipe{ // this is receiver pipe, i.e. opposite direction
		In:  wrongAgentDid, // start with last route
		Out: didIn,         // and now reiver is original sender
	}

	wrongFwBytes, _, err := wrongUnpackPipe.Unpack(route2bytes)
	assert.Error(err)
	assert.SNil(wrongFwBytes)

	// Unpack next forward message with first routing key
	//	route1FwdMsg := aries.PayloadCreator.NewFromData(route1FwBytes).MsgHdr().FieldObj().(*common.Forward)
	//	route1Bytes := route1FwdMsg.Msg
	//	route1Keys, err := getRecipientKeys(route1Bytes)
	//	assert.NoError( err)
	//	assert.SLen( route1Keys, 2)
	//	assert.Equal( didRoute1.VerKey(), route1Keys[0])
	//	assert.Equal( didRoute1.VerKey(), route1FwdMsg.To)

	//	// Unpack final (auth-crypted) message with destination key
	//	dstBytes, _, err := dstUnpackPipe.Unpack(dstPackedBytes)
	//	assert.NoError( err)
	//	dstKeys, err := getRecipientKeysFromBytes(dstPackedBytes)
	//	assert.NoError( err)
	//	assert.That( len(dstFwdKeys) == 1)
	//	assert.Equal( didOut.VerKey(), dstKeys[0])
	//
	//	dstMsg := aries.PayloadCreator.NewFromData(dstBytes)
	//	assert.That( dstMsg.MsgHdr().Type() == pltype.AriesConnectionRequest)
	//	assert.That( dstMsg.MsgHdr().ID() == plID)
	//	assert.That( dstMsg.MsgHdr().FieldObj().(*v0.Request).Label == "test")
}
