package common

import (
	"testing"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/stretchr/testify/assert"
)

var forwardJSON = `
  {
    "@type": "https://didcomm.org/routing/1.0/forward",
    "@id": "54ad1a63-29bd-4a59-abed-1c5b1026e6fd",
    "to": "did:sov:1234abcd#4"
  }`

func TestForward_ReadJSON(t *testing.T) {
	ipl := aries.PayloadCreator.NewFromData([]byte(forwardJSON))

	assert.Equal(t, "54ad1a63-29bd-4a59-abed-1c5b1026e6fd", ipl.ID())

	msg, ok := ipl.MsgHdr().FieldObj().(*Forward)
	assert.True(t, ok)
	assert.Equal(t, "did:sov:1234abcd#4", msg.To)
}
