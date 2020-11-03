package server

import (
	"github.com/findy-network/findy-agent-api/grpc/ops"
)

type agencyService struct {
	ops.UnimplementedAgencyServer
}

func (a agencyService) PSMHook(hook *ops.DataHook, server ops.Agency_PSMHookServer) error {
	panic("implement me")
}
