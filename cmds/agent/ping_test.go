package agent

import (
	"testing"

	"github.com/findy-network/findy-agent/cmds"
	"github.com/stretchr/testify/assert"
)

func TestCmd_Build(t *testing.T) {
	invalid := PingCmd{Cmd: cmds.Cmd{
		WalletName: "",
		WalletKey:  "test-key",
	}}
	err := invalid.Validate()
	assert.Error(t, err)

	c := PingCmd{Cmd: cmds.Cmd{
		WalletName: "test-wallet",
		WalletKey:  "test-key",
	}}
	err = c.Validate()
	assert.Error(t, err) // wallet not exist
}
