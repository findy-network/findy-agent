package bus

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/golang/glog"
)

type AgentKeyType struct {
	AgentDID string
	ClientID string
}

type AgentStateChan chan AgentNotify

type AgentNotify struct {
	AgentKeyType
	ID               string
	PID              string
	NotificationType string
	ConnectionID     string
	ProtocolID       string
	ProtocolFamily   string
	Timestamp        int64
	Initiator        bool
	*IssuePropose
	*ProofVerify
}

type IssuePropose struct {
	CredDefID  string
	ValuesJSON string
}

type ProofVerify struct {
	Attrs []didcomm.ProofValue
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

	glog.V(3).Infoln(key.AgentDID, " notify add for:", key.ClientID)
	AgentMaps[m].agentStationMap[key] = make(AgentStateChan, 1)
	return AgentMaps[m].agentStationMap[key]
}

func (m mapIndex) AgentRmListener(key AgentKeyType) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	glog.V(3).Infoln(key.AgentDID, " notify rm for:", key.ClientID)
	delete(AgentMaps[m].agentStationMap, key)
}

func (m mapIndex) AgentBroadcast(state AgentNotify) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	key := state.AgentKeyType
	for k, ch := range AgentMaps[m].agentStationMap {
		if key.AgentDID == k.AgentDID {
			glog.V(3).Infoln(key.AgentDID, " agent notify:", k.ClientID)
			state.AgentKeyType.ClientID = k.ClientID
			ch <- state
		}
	}
}
