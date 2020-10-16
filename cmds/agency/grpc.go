package agency

import (
	"github.com/findy-network/findy-agent/plugins"
	"github.com/golang/glog"
)

func StartGrpcServer() {
	if plugin := plugins.GetPlugin("GRPC"); plugin != nil {
		go plugin.Run()
	} else {
		glog.Warningf("---------\n%s\n---------",
			"grpc plugin is not installed")
	}
}
