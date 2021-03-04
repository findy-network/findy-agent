package server

import (
	"context"
	"errors"
	"flag"
	"fmt"

	agency "github.com/findy-network/findy-agent-api/grpc/ops"
	agencyServer "github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type devOpsServer struct {
	agency.UnimplementedDevOpsServer
	Root string
}

func (d devOpsServer) Enter(ctx context.Context, cmd *agency.Cmd) (cr *agency.CmdReturn, err error) {
	defer err2.Return(&err)

	user := jwt.User(ctx)
	if user != d.Root {
		return &agency.CmdReturn{Type: cmd.Type}, errors.New("access right")
	}

	glog.V(3).Infoln("dev ops cmd", cmd.Type)
	cmdReturn := &agency.CmdReturn{Type: cmd.Type}

	switch cmd.Type {
	case agency.Cmd_PING:
		response := fmt.Sprintf("%s, ping ok", utils.Settings.VersionInfo())
		cmdReturn.Response = &agency.CmdReturn_Ping{Ping: response}
	case agency.Cmd_LOGGING:
		err2.Check(flag.Set("v", cmd.GetLogging()))
	case agency.Cmd_COUNT:
		response := fmt.Sprintf("%d/%d cloud agents",
			agencyServer.HandlerCount(), agencyServer.SeedHandlerCount())
		cmdReturn.Response = &agency.CmdReturn_Count{Count: response}
	}
	return cmdReturn, nil
}
