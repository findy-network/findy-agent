package grpc

import (
	"github.com/findy-network/findy-agent/grpc/server"
	"github.com/findy-network/findy-agent/plugins"
	"github.com/findy-network/findy-grpc/rpc"
	"github.com/golang/glog"
)

func init() {
	plugins.AddPlugin("GRPC", &RpcServer)
}

type PluginServer struct {
	Port      int
	TlsPath   string
	JWTSecret string
}

var RpcServer PluginServer

func (s *PluginServer) Run() {
	glog.V(1).Infoln("===== initializing grpc server")
	server.Serve(&rpc.ServerCfg{
		Port:      s.Port,
		PKI:       rpc.LoadPKI(s.TlsPath),
		TestLis:   nil,
		JWTSecret: s.JWTSecret,
	})
}

func (s *PluginServer) Init(port int, tlsPath, jwtSecret string) {
	glog.V(1).Infof("init plugin with port(%d) tls (%s)", port, tlsPath)
	s.Port = port
	s.TlsPath = tlsPath
	s.JWTSecret = jwtSecret
}
