package agency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd_Build(t *testing.T) {
	invalid := Cmd{
		WalletName: "test",
		WalletPwd:  "test-key",
	}
	err := invalid.Validate()
	assert.Error(t, err)

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
	assert.NoError(t, err)
}
