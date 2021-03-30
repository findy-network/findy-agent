package grpc

import (
	"github.com/findy-network/findy-agent/grpc/server"
	"github.com/findy-network/findy-agent/plugins"
	"github.com/findy-network/findy-common-go/rpc"
	"github.com/golang/glog"
)

func init() {
	plugins.AddPlugin("GRPC", &RpcServer)
}

type PluginServer struct {
	UseTls    bool
	Port      int
	TlsPath   string
	JWTSecret string
}

var RpcServer PluginServer

func (s *PluginServer) Run() {
	glog.V(1).Infoln("===== initializing grpc server")

	var pki *rpc.PKI
	if s.UseTls {
		pki = rpc.LoadPKI(s.TlsPath)
	}

	server.Serve(&rpc.ServerCfg{
		Port:      s.Port,
		PKI:       pki,
		TestLis:   nil,
		JWTSecret: s.JWTSecret,
	})
}

func (s *PluginServer) Init(useTls bool, port int, tlsPath, jwtSecret string) {
	glog.V(1).Infof("init plugin with port(%d) tls (%v %s)", port, useTls, tlsPath)
	s.UseTls = useTls
	s.Port = port
	s.TlsPath = tlsPath
	s.JWTSecret = jwtSecret
}
