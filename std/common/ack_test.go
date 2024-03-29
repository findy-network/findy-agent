package common

import (
	"testing"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/lainio/err2/assert"
)

var ackJSON = `
  {
    "@type": "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/issue-credential/1.0/ack",
    "@id": "3eb5fd37-48ac-4767-8cce-07ab5bbe9097",
    "~thread": { "thid": "3dc323d4-17ec-4a4a-9d3a-c903e94d253b" },
    "status": "OK"
  }`

func TestAck_ReadJSON(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	ipl := aries.PayloadCreator.NewFromData([]byte(ackJSON))

	assert.Equal("3eb5fd37-48ac-4767-8cce-07ab5bbe9097", ipl.ID())
	assert.Equal("3dc323d4-17ec-4a4a-9d3a-c903e94d253b", ipl.ThreadID())

	msg, ok := ipl.MsgHdr().FieldObj().(*Ack)
	assert.That(ok)
	assert.NotEmpty(msg.Status)
}
