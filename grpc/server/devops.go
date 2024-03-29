package server

import (
	"context"
	"errors"
	"flag"
	"fmt"

	agencyServer "github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/utils"
	agency "github.com/findy-network/findy-common-go/grpc/ops/v1"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type devOpsServer struct {
	agency.UnimplementedDevOpsServiceServer
	Root string
}

func (d devOpsServer) Enter(ctx context.Context, cmd *agency.Cmd) (cr *agency.CmdReturn, err error) {
	defer err2.Handle(&err)

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
		try.To(flag.Set("v", cmd.GetLogging()))
	case agency.Cmd_COUNT:
		response := fmt.Sprintf("%d/%d cloud agents",
			agencyServer.HandlerCount(), agencyServer.SeedHandlerCount())
		cmdReturn.Response = &agency.CmdReturn_Count{Count: response}
	}
	return cmdReturn, nil
}
