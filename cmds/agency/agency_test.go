package agency

import (
	"testing"

	"github.com/lainio/err2/assert"
)

func TestCmd_BuildNOK(t *testing.T) {
	assert.PushTester(t, assert.Production)
	defer assert.PopTester()

	invalid := Cmd{
		WalletName: "test",
		WalletPwd:  "test-key",
	}
	err := invalid.Validate()
	assert.SetDefault(assert.Test)
	assert.Error(err)
}

func TestCmd_BuildOK(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	c := Cmd{
		PoolName:          "tste",
		WalletName:        "test-wallet",
		WalletPwd:         "test-key",
		StewardSeed:       "",
		ServiceName:       "findy2",
		HostAddr:          "localhost",
		HostPort:          80,
		ServerPort:        80,
		StewardDid:        "did",
		HandshakeRegister: "findy.json",
		PsmDB:             "psm.bolt",
	}
	err := c.Validate()
	assert.NoError(err)
}
