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
	psmCleanup
)

// todo: should we add the buffer here? now we have only channel but if there is
// no cannel we would add least have a resend buffer? BUT if it's at the upper
// level we don't need to wonder when we will create the buffer. It should be
// there anyhow.
type agentStationMap map[AgentKeyType]AgentStateChan

type buf []*AgentNotify
type buffer struct {
	buf
	sync.Mutex
}

type agentStation struct {
	agentStationMap
	sync.Mutex

	// buffer stores notifications if no one listens
	buffer
}

var (
	AgentMaps = [...]agentStation{
		{
			agentStationMap: make(agentStationMap),
			buffer: buffer{
				buf: make(buf, 0, 12), // Agent Actions are resend now.
			},
		},
		{
			agentStationMap: make(agentStationMap),
			buffer:          buffer{
				// buf: make(buf, 0, 12),
			},
		},
		{
			agentStationMap: make(agentStationMap),
			buffer:          buffer{
				// buf: make(buf, 0, 12),
			},
		},
	}

	WantAllAgentActions  mapIndex = agentListen
	WantAllAgencyActions mapIndex = agencyListen
	WantAllPSMCleanup    mapIndex = psmCleanup
)

func (m mapIndex) AgentAddListener(key AgentKeyType) AgentStateChan {
	c := make(AgentStateChan, 1)
	AgentMaps[m].Lock()
	AgentMaps[m].agentStationMap[key] = c
	AgentMaps[m].Unlock()

	glog.V(3).Infoln(key.AgentDID, " notify add for:", key.ClientID)

	if m == WantAllAgentActions {
		go m.checkBuffered()
	}
	return c
}

// checkBuffered sends all buffered notifications to listeners and reset the
// buffer. TODO: make buffer persistent.
func (m mapIndex) checkBuffered() {
	AgentMaps[m].buffer.Lock()
	defer AgentMaps[m].buffer.Unlock()

	for _, notif := range AgentMaps[m].buffer.buf { // nil is checked by Go
		glog.V(3).Infoln("+++++++++++++++++ broadcasting buffered notification",
			notif.NotificationType)

		// broadcast needs outside locking
		AgentMaps[m].Lock()
		m.broadcast(notif)
		AgentMaps[m].Unlock()
	}
	AgentMaps[m].buffer.buf = make(buf, 0, 12)
}

// AgentRmListener removes the listener.
func (m mapIndex) AgentRmListener(key AgentKeyType) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	glog.V(3).Infoln(key.AgentDID, " notify rm for:", key.ClientID)
	ch, ok := AgentMaps[m].agentStationMap[key]
	if ok {
		close(ch)
		delete(AgentMaps[m].agentStationMap, key)
	}
}

// AgentBroadcast broadcasts the notification.
// TODO: add persitency here
func (m mapIndex) AgentBroadcast(state AgentNotify) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	if len(AgentMaps[m].agentStationMap) == 0 { //
		glog.V(1).Infoln("+++++++++++++++++++++\nthere are no one to listen us!")
		// Save the notification for future broadcasting

		// TODO: build the resender here who do the job
		// the DB could be one bucket where key is DID and notifications are
		// kept in the slices. Remember ActionsTypes or own buckets per.

		// first implementation just saves notifications to buffer, no persitency
		go m.saveToResendBuf(state)
		return
	}
	m.broadcast(&state)
}

func (m mapIndex) saveToResendBuf(state AgentNotify) {
	AgentMaps[m].buffer.Lock()
	defer AgentMaps[m].buffer.Unlock()

	if AgentMaps[m].buffer.buf != nil {
		AgentMaps[m].buffer.buf = append(AgentMaps[m].buffer.buf, &state)
	}
}

// broadcast broadcasts notification to listeners. Note! it doesn't lock
// the maps.
func (m mapIndex) broadcast(state *AgentNotify) {
	broadcastKey := state.AgentKeyType
	for listenKey, ch := range AgentMaps[m].agentStationMap {
		hit := broadcastKey.AgentDID == listenKey.AgentDID ||
			listenKey.AgentDID == AllAgents
		if hit {
			glog.V(3).Infoln(broadcastKey.AgentDID,
				"agent notify:", listenKey.ClientID)
			state.ClientID = listenKey.ClientID
			ch <- *state
		}
	}
}
