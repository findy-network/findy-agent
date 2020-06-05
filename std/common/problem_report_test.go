package common

import (
	"testing"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/stretchr/testify/assert"
)

var json = `
{
  "@type": "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/notification/1.0/problem-report",
  "@id": "8e59230b-47e4-4abb-a5cc-28d1b09f0e96",
  "~thread": {
    "thid": "8225993b-73f9-404c-804b-139bd03893dc"
  },
  "explain-ltxt": "Error deserializing message: CredentialAck schema validation failed"
}`

func TestProblemReport_ReadJSON(t *testing.T) {
	ipl := aries.PayloadCreator.NewFromData([]byte(json))

	assert.Equal(t, "8e59230b-47e4-4abb-a5cc-28d1b09f0e96", ipl.ID())
	assert.Equal(t, "8225993b-73f9-404c-804b-139bd03893dc", ipl.ThreadID())

	msg, ok := ipl.MsgHdr().FieldObj().(*ProblemReport)
	assert.True(t, ok)
	assert.NotEmpty(t, msg.ExplainLongTxt)
}
