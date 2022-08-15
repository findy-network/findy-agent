package agency

import (
	"testing"

	"github.com/lainio/err2/assert"
)

func TestCmd_Build(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	invalid := Cmd{
		WalletName: "test",
		WalletPwd:  "test-key",
	}
	err := invalid.Validate()
	assert.Error(err)

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
	err = c.Validate()
	assert.NoError(err)
}
