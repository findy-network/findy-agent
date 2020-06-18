package sa

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListCmd_Exec(t *testing.T) {
	cmd := ListCmd{}
	err := cmd.Validate()
	assert.NoError(t, err)
	r, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
	result, ok := r.(*ListResult)
	assert.True(t, ok)
	assert.NotEmpty(t, result.Implementations)
}
