package key

import (
	"os"
	"testing"

	"github.com/lainio/err2/assert"
)

func TestCreateCmd_Exec(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	cmd := CreateCmd{Seed: "00000000000000000000thisisa_test"}
	err := cmd.Validate()
	assert.NoError(err)
	_, err = cmd.Exec(os.Stdout)
	assert.NoError(err)
}
