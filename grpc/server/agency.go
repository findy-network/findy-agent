package server

import (
	"github.com/findy-network/findy-agent-api/grpc/agency"
)

type agencyService struct {
	agency.UnimplementedAgencyServer
}

func (a agencyService) PSMHook(hook *agency.DataHook, server agency.Agency_PSMHookServer) error {
	panic("implement me")
}
