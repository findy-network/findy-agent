package bus

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/golang/glog"
)

const AllAgents = "*"

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
	agencyListen
)

type agentStationMap map[AgentKeyType]AgentStateChan
type agentLockMap struct {
	agentStationMap
	sync.Mutex
}

var AgentMaps = [...]agentLockMap{
	{agentStationMap: make(agentStationMap)},
	{agentStationMap: make(agentStationMap)},
}

var WantAllAgentActions mapIndex = agentListen
var WantAllAgencyActions mapIndex = agencyListen

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

	broadcastKey := state.AgentKeyType
	for listenKey, ch := range AgentMaps[m].agentStationMap {
		hit := broadcastKey.AgentDID == listenKey.AgentDID ||
			listenKey.AgentDID == AllAgents
		if hit {
			glog.V(3).Infoln(broadcastKey.AgentDID,
				"agent notify:", listenKey.ClientID)
			state.AgentKeyType.ClientID = listenKey.ClientID
			ch <- state
		}
	}
}
