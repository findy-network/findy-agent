package common

import (
	"testing"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/lainio/err2/assert"
)

var forwardJSON = `
  {
    "@type": "https://didcomm.org/routing/1.0/forward",
    "@id": "54ad1a63-29bd-4a59-abed-1c5b1026e6fd",
    "to": "did:sov:1234abcd#4"
  }`

func TestForward_ReadJSON(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	ipl := aries.PayloadCreator.NewFromData([]byte(forwardJSON))

	assert.Equal("54ad1a63-29bd-4a59-abed-1c5b1026e6fd", ipl.ID())

	msg, ok := ipl.MsgHdr().FieldObj().(*Forward)
	assert.That(ok)
	assert.Equal("did:sov:1234abcd#4", msg.To)
}
