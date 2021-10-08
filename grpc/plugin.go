package grpc

import (
	"github.com/findy-network/findy-agent/grpc/server"
	"github.com/findy-network/findy-agent/plugins"
	"github.com/findy-network/findy-common-go/rpc"
	"github.com/golang/glog"
)

func init() {
	plugins.AddPlugin("GRPC", &RPCServer)
}

type PluginServer struct {
	UseTLS    bool
	Port      int
	TLSPath   string
	JWTSecret string
}

var RPCServer PluginServer

func (s *PluginServer) Run() {
	glog.V(1).Infoln("===== initializing grpc server")

	var pki *rpc.PKI
	if s.UseTLS {
		pki = rpc.LoadPKI(s.TLSPath)
	}

	server.Serve(&rpc.ServerCfg{
		Port:      s.Port,
		PKI:       pki,
		TestLis:   nil,
		JWTSecret: s.JWTSecret,
	})
}

func (s *PluginServer) Init(useTLS bool, port int, tlsPath, jwtSecret string) {
	glog.V(1).Infof("init plugin with port(%d) tls (%v %s)", port, useTLS, tlsPath)
	s.UseTLS = useTLS
	s.Port = port
	s.TLSPath = tlsPath
	s.JWTSecret = jwtSecret
}
