package bus

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/golang/glog"
)

type KeyType = psm.StateKey
type Ready chan bool
type StateChan chan psm.SubState

type Station struct {
}

type mapIndex int

const (
	allStates = 0 + iota
	userActions
)

type stationMap map[KeyType]StateChan
type lockMap struct {
	stationMap
	sync.Mutex
}

var maps = [...]lockMap{{stationMap: make(stationMap)}, {stationMap: make(stationMap)}}
var WantAll mapIndex = allStates
var WantUserActions mapIndex = userActions

func (m mapIndex) AddListener(key KeyType) StateChan {
	maps[m].Lock()
	defer maps[m].Unlock()

	maps[m].stationMap[key] = make(StateChan)
	return maps[m].stationMap[key]
}

func (m mapIndex) RmListener(key KeyType) {
	maps[m].Lock()
	defer maps[m].Unlock()
	delete(maps[m].stationMap, key)
}

func (m mapIndex) Broadcast(key KeyType, state psm.SubState) {
	// We need to unlock as soon as possible that we don't keep lock when
	// blocking channel send at the end.
	maps[m].Lock()

	c, ok := maps[m].stationMap[key]
	if !ok {
		maps[m].Unlock() // Manual unlock needed, see below
		return
	}
	maps[m].Unlock() // Important! Leve lock before writing channel

	c <- state
}

func BroadcastReboot() {
	for i := range maps {
		for _, c := range maps[i].stationMap {
			glog.V(1).Infoln("signaling reboot for listener")
			c <- psm.SystemReboot
		}
	}
	for i := range agentMaps {
		for _, c := range agentMaps[i].agentStationMap {
			glog.V(1).Infoln("signaling reboot for listener")
			c <- *NewRebootAgentNotify()
		}
	}
}
