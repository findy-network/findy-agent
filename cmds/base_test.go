package cmds

import (
	"testing"

	"github.com/lainio/err2/assert"
)

func TestValidateTime(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	err := ValidateTime("21:45")
	assert.NoError(err)
	err = ValidateTime("01:37:48")
	assert.NoError(err)
	err = ValidateTime("24:00:00")
	assert.Error(err)
}
