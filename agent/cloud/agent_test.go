package cloud

import (
	"testing"

	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/stretchr/testify/require"
)

func TestCAEndp(t *testing.T) {
	a := Agent{
		DIDAgent: ssi.DIDAgent{
			Type: ssi.Worker,
		},
		myDID: ssi.NewDid("did", "verkey"),
	}
	connID := "connID"
	serviceName := "serviceName"
	hostAddr := "hostAddr"

	utils.Settings.SetServiceName(serviceName)
	utils.Settings.SetHostAddr(hostAddr)

	endpoint := a.CAEndp(connID)
	require.Equal(t, endp.Addr{
		ID:        0,
		Service:   serviceName,
		PlRcvr:    a.myDID.Did(),
		MsgRcvr:   a.myDID.Did(),
		ConnID:    connID,
		EdgeToken: "",
		BasePath:  hostAddr,
		VerKey:    a.myDID.VerKey(),
	}, *endpoint)
}
