package basicmessage

import (
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-common-go/dto"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/lainio/err2/assert"
)

var timeJSON = "{\"sent_time\":\"2020-03-20 12:06:36.225671Z\"}"
var timeJSONRFC3339 = "{\"sent_time\":\"2022-09-30T12:31:05.923762Z\"}"

var mbJSON = `{
    "@type": "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/basicmessage/1.0/message",
    "@id": "a70a5db1-0b35-41d2-a602-e355ec4df67f",
    "content": "test",
    "sent_time": "2020-01-20 12:06:36.225671Z"
  }`

func TestNewTimeField(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	assert.PushTester(t)
	defer assert.PopTester()

	var testMsg Basicmessage
	dto.FromJSON([]byte(timeJSON), &testMsg)
	timeValue := testMsg.SentTime

	assert.INotNil(timeValue)
	assert.Equal(timeValue.Year(), 2020)
	assert.Equal(timeValue.Month(), time.March)
	assert.Equal(timeValue.Day(), 20)

}

func TestNewTimeFieldRFC3339(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	assert.PushTester(t)
	defer assert.PopTester()

	var testMsg Basicmessage
	dto.FromJSON([]byte(timeJSONRFC3339), &testMsg)
	timeValue := testMsg.SentTime

	assert.INotNil(timeValue)
	assert.Equal(timeValue.Year(), 2022)
	assert.Equal(timeValue.Month(), time.September)
	assert.Equal(timeValue.Day(), 30)
}

func TestNewBasicmessage(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	assert.PushTester(t)
	defer assert.PopTester()

	ipl := aries.PayloadCreator.NewFromData([]byte(mbJSON))

	assert.Equal("a70a5db1-0b35-41d2-a602-e355ec4df67f", ipl.ID())
	assert.Equal("a70a5db1-0b35-41d2-a602-e355ec4df67f", ipl.ThreadID())

	msg, ok := ipl.MsgHdr().FieldObj().(*Basicmessage)
	assert.That(ok)
	assert.NotEmpty(msg.Content)
}

func TestBasicMessage_MsgPingPong(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	assert.PushTester(t)
	defer assert.PopTester()

	//sencondID := "2nd id"
	firstMsgID := "1st id"

	send1, ok := aries.MsgCreator.Create(didcomm.MsgInit{
		AID:  firstMsgID,
		Type: pltype.BasicMessageSend,
		Info: "hello",
		//Thread: decorator.NewThread(firstMsgID, inviteID),
	}).(*Impl)
	assert.That(ok)

	data := send1.JSON()
	println(string(data))

	ipl := aries.PayloadCreator.NewFromData(data)
	msg, msgOK := ipl.FieldObj().(*Impl)
	assert.That(msgOK)
	assert.Equal("hello", msg.Content)
	//assert.Equal( offer.Thread().ID, firstMsgID)
}
