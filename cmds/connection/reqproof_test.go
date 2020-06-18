package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReqProofCmd_Exec(t *testing.T) {
	attrs, err := parseProofAttrs(`[{"name":"email","creddefid":"TEST_CRED_DEF_ID"}]`)
	assert.NoError(t, err)
	assert.Equal(t, "email", attrs[0].Name)
	assert.Equal(t, "TEST_CRED_DEF_ID", attrs[0].CredDefID)
}
