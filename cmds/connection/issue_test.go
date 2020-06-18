package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssueCmd_Exec(t *testing.T) {
	attrs, err := parseAttrs(`[{"name":"email","value":"test@email.com"}]`)
	assert.NoError(t, err)
	assert.Equal(t, "email", attrs[0].Name)
	assert.Equal(t, "test@email.com", attrs[0].Value)
}
