package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssueCmd_Exec(t *testing.T) {
	_, err := parseAttrs(`[{"name":"email","value":"test@email.com"}]`)
	assert.NoError(t, err)
}
