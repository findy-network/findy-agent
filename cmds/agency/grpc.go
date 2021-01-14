package agency

import (
	"github.com/findy-network/findy-agent/grpc"
	"github.com/findy-network/findy-agent/plugins"
	"github.com/golang/glog"
)

func StartGrpcServer(port int, tlsCertPath, jwtSecret string) {
	if plugin := plugins.GetPlugin("GRPC"); plugin != nil {
		p := plugin.(*grpc.PluginServer)
		p.Init(port, tlsCertPath, jwtSecret)
		go plugin.Run()
	} else {
		glog.Warningf("\n---------\n%s\n---------",
			"grpc plugin is not installed")
	}
}
