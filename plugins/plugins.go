/*
Package plugins is general purpose package for findy-agent to register addons
for agency as a Go code without direct dependency to the implementation and to
the repo. For example, gRPC server code is currently implemented in separate git
repo to allow pure incremental development of it. The agency cmd that invokes
the server only refers to it by name, so no package direct import is needed. The
gRPC server plugin installs itself at the build time thru findy-service-tool,
which registers the specific plugin i.e. gRPC service.
*/
package plugins

import (
	"sync"

	"github.com/golang/glog"
)

// Plugin is a plugin interface for addon ledger implementations.
type Plugin interface {
	Run()
}

type RegisteredPlugins map[string]Plugin

type Map struct {
	sync.RWMutex
	ory RegisteredPlugins
}

var mem = Map{ory: make(RegisteredPlugins)}

func AddPlugin(name string, plugin Plugin) {
	mem.Lock()
	defer mem.Unlock()
	if _, ok := mem.ory[name]; ok {
		panic("plugin with the name:" + name + "already exists")
	}
	mem.ory[name] = plugin
}

func GetPlugin(name string) Plugin {
	mem.RLock()
	defer mem.RUnlock()
	glog.V(3).Infoln("getting plugin", name)
	return mem.ory[name]
}
