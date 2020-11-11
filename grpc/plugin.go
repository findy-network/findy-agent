package grpc

import (
	"github.com/findy-network/findy-agent/grpc/server"
	"github.com/findy-network/findy-agent/plugins"
	"github.com/golang/glog"
)

func init() {
	plugins.AddPlugin("GRPC", &RpcServer)
}

type PluginServer struct {
	Port    int
	TlsPath string
}

var RpcServer PluginServer

func (s *PluginServer) Run() {
	server.Serve(nil)
}

func (s *PluginServer) Init(port int, tlsPath string) {
	glog.V(1).Infof("init plugin with port(%d) tls (%s)", port, tlsPath)
	s.Port = port
	s.TlsPath = tlsPath
}
