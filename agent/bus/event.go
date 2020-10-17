package bus

import (
	"sync"
)

type AgentKeyType struct {
	AgentDID string
	ClientID string
}

type AgentStateChan chan AgentNotify

type AgentNotify struct {
	AgentKeyType
	NotificationType string
	ConnectionID     string
	ProtocolID       string
	ProtocolFamily   string
	TimestampMs      uint64
}

const (
	agentListen = 0 + iota
)

type agentStationMap map[AgentKeyType]AgentStateChan
type agentLockMap struct {
	agentStationMap
	sync.Mutex
}

var AgentMaps = [...]agentLockMap{{agentStationMap: make(agentStationMap)}}

var WantAllAgentActions mapIndex = agentListen

func (m mapIndex) AgentAddListener(key AgentKeyType) AgentStateChan {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	AgentMaps[m].agentStationMap[key] = make(AgentStateChan, 1)
	return AgentMaps[m].agentStationMap[key]
}

func (m mapIndex) AgentRmListener(key AgentKeyType) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()
	delete(AgentMaps[m].agentStationMap, key)
}

func (m mapIndex) AgentBroadcast(state AgentNotify) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	key := state.AgentKeyType
	for k, ch := range AgentMaps[m].agentStationMap {
		if key.AgentDID == k.AgentDID {
			ch <- state
		}
	}
}
