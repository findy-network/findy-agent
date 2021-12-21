package sec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/std/common"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-agent/std/did"
	"github.com/findy-network/findy-agent/std/didexchange"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/lainio/err2"
	"github.com/stretchr/testify/assert"
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

func setUp() {
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
	err2.Return(&err)

	protData := err2.Bytes.Try(utils.DecodeB64(msg["protected"].(string)))

	data := protected{}
	err2.Check(json.Unmarshal(protData, &data))

	keys = make([]string, 0)
	for _, recipient := range data.Recipients {
		keys = append(keys, recipient.Header.Kid)
	}
	return keys, nil
}

func TestPipe_pack(t *testing.T) {
	// Create test wallet and routing keys
	walletID := fmt.Sprintf("pipe-test-agent-%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()
	a := ssi.DIDAgent{}
	a.OpenWallet(*aw)
	didIn := a.CreateDID("")
	didOut := a.CreateDID("")
	didRoute1 := a.CreateDID("")
	didRoute2 := a.CreateDID("")

	// Packing pipe with two routing keys
	packPipe := NewPipeByVerkey(didIn, didOut.VerKey(), []string{didRoute1.VerKey(), didRoute2.VerKey()})

	// Simulate actual aries message
	plID := utils.UUID()
	msg := didexchange.NewRequest(&didexchange.Request{
		Label: "test",
		Connection: &didexchange.Connection{
			DID: didIn.Did(),
			DIDDoc: did.NewDoc(didIn, service.Addr{
				Endp: "http://example.com",
				Key:  didIn.VerKey(),
			}),
		},
		Thread: &decorator.Thread{ID: utils.UUID()},
	})
	pl := aries.PayloadCreator.NewMsg(plID, pltype.AriesConnectionRequest, msg)

	// Pack message
	route2bytes, _, err := packPipe.Pack(pl.JSON())
	assert.Nil(t, err)
	assert.True(t, len(route2bytes) > 0)

	// Unpack forward message with last routing key
	route2Keys, err := getRecipientKeysFromBytes(route2bytes)
	assert.Nil(t, err)
	assert.True(t, len(route2Keys) == 1)
	assert.Equal(t, didRoute2.VerKey(), route2Keys[0])

	route1UnpackPipe := NewPipeByVerkey(didRoute2, didIn.VerKey(), []string{})
	route1FwBytes, _, err := route1UnpackPipe.Unpack(route2bytes)
	assert.Nil(t, err)

	// Unpack next forward message with first routing key
	route1FwdMsg := aries.PayloadCreator.NewFromData(route1FwBytes).MsgHdr().FieldObj().(*common.Forward)
	route1Bytes := route1FwdMsg.Msg
	route1Keys, err := getRecipientKeys(route1Bytes)
	assert.Nil(t, err)
	assert.True(t, len(route2Keys) == 1)
	assert.Equal(t, didRoute1.VerKey(), route1Keys[0])
	assert.Equal(t, didRoute1.VerKey(), route1FwdMsg.To)

	dstUnpackPipe := NewPipeByVerkey(didOut, didIn.VerKey(), []string{})
	dstFwBytes, _, err := dstUnpackPipe.Unpack(dto.ToJSONBytes(route1Bytes))
	assert.Nil(t, err)

	// Unpack final (anon-crypted) forward message with destination key
	dstFwdMsg := aries.PayloadCreator.NewFromData(dstFwBytes).MsgHdr().FieldObj().(*common.Forward)
	dstPackedBytes := dto.ToJSONBytes(dstFwdMsg.Msg)
	dstFwdKeys, err := getRecipientKeys(dstFwdMsg.Msg)
	assert.Nil(t, err)
	assert.True(t, len(dstFwdKeys) == 1)
	assert.Equal(t, didOut.VerKey(), dstFwdKeys[0])
	assert.Equal(t, didOut.VerKey(), dstFwdMsg.To)

	// Unpack final (auth-crypted) message with destination key
	dstBytes, _, err := dstUnpackPipe.Unpack(dstPackedBytes)
	assert.Nil(t, err)
	dstKeys, err := getRecipientKeysFromBytes(dstPackedBytes)
	assert.Nil(t, err)
	assert.True(t, len(dstFwdKeys) == 1)
	assert.Equal(t, didOut.VerKey(), dstKeys[0])

	dstMsg := aries.PayloadCreator.NewFromData(dstBytes)
	assert.True(t, dstMsg.MsgHdr().Type() == pltype.AriesConnectionRequest)
	assert.True(t, dstMsg.MsgHdr().ID() == plID)
	assert.True(t, dstMsg.MsgHdr().FieldObj().(*didexchange.Request).Label == "test")
}
