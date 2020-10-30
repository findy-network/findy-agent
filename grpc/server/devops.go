package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/findy-network/findy-agent-api/grpc/agency"
	agencyServer "github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/utils"
	agencyCmd "github.com/findy-network/findy-agent/cmds/agency"
	"github.com/findy-network/findy-grpc/jwt"
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
		agencyCmd.ParseLoggingArgs(cmd.GetLogging())
		//response = fmt.Sprintf("logging = %s", cmd.GetLogging())
	case agency.Cmd_COUNT:
		response := fmt.Sprintf("%d/%d cloud agents",
			agencyServer.SeedHandlerCount(), agencyServer.HandlerCount())
		cmdReturn.Response = &agency.CmdReturn_Ping{Ping: response}
	}
	return cmdReturn, nil
}
