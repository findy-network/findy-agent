package grpc

import (
	"github.com/findy-network/findy-agent/grpc/server"
	"github.com/findy-network/findy-agent/plugins"
)

func init() {
	plugins.AddPlugin("GRPC", grpcServer)
}

type pluginServer struct{}

var grpcServer pluginServer

func (s pluginServer) Run() {
	server.Serve()
}
