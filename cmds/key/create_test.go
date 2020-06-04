package key

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCmd_Exec(t *testing.T) {
	cmd := CreateCmd{Seed: "00000000000000000000thisisa_test"}
	err := cmd.Validate()
	assert.NoError(t, err)
	_, err = cmd.Exec(os.Stdout)
	assert.NoError(t, err)
}
