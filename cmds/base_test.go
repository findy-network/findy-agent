package cmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTime(t *testing.T) {
	err := ValidateTime("21:45")
	assert.NoError(t, err)
	err = ValidateTime("01:37:48")
	assert.NoError(t, err)
	err = ValidateTime("24:00:00")
	assert.Error(t, err)
}
