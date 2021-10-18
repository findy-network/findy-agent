package bus

import (
	"container/list"
	"sync"

	"github.com/findy-network/findy-agent/agent/didcomm"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	"github.com/golang/glog"
	"github.com/lainio/err2/assert"
)

const AllAgents = "*"

type AgentKeyType struct {
	AgentDID string
	ClientID string
}

func (k AgentKeyType) String() string {
	return "AgentKey:" + k.AgentDID + "|" + k.ClientID
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
	UserActionType   string
	Role             pb.Protocol_Role
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

// TODO: should we add the buffer here? now we have only channel but if there is
// no cannel we would add least have a resend buffer? BUT if it's at the upper
// level we don't need to wonder when we will create the buffer. It should be
// there anyhow.
type agentStationMap map[AgentKeyType]AgentStateChan

type buffer struct {
	buf *list.List
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
				buf: list.New(), // Agent Actions are resend now.
			},
		},
		{
			agentStationMap: make(agentStationMap),
			buffer:          buffer{
				// buf:
			},
		},
		{
			agentStationMap: make(agentStationMap),
			buffer:          buffer{
				// buf:
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
	_, alreadyExists := AgentMaps[m].agentStationMap[key]
	assert.D.Truef(!alreadyExists, "key: %s, already exists", key)

	AgentMaps[m].agentStationMap[key] = c
	AgentMaps[m].Unlock()

	glog.V(4).Infoln(key.AgentDID, "notify ADD for: ", key.ClientID)

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

	l := AgentMaps[m].buffer.buf

	// using linked list this way it's safe to remove items during iteration
	for e := l.Front(); e != nil; {

		notif := e.Value.(*AgentNotify)
		glog.V(14).Infoln(notif.ClientID,
			"+++ trying to broadcast buffered notif for:\n",
			notif.AgentKeyType)

		// save current element and iterate for safe removal during loop
		old := e
		e = e.Next()

		// broadcast sends notifications only those agent's ctrls that listen
		if m.lockedBroadcast(notif) {
			// now it's safe to remove during the for loop
			glog.V(14).Infoln("removing: ", notif.ClientID)
			l.Remove(old)
		}
	}
	glog.V(3).Infoln("checkBuffered done")
}

// AgentRmListener removes the listener.
func (m mapIndex) AgentRmListener(key AgentKeyType) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	if glog.V(4) {
		glog.Infoln(key.AgentDID, " notify RM for:", key.ClientID)
	}
	ch, ok := AgentMaps[m].agentStationMap[key]
	if ok {
		close(ch)
		delete(AgentMaps[m].agentStationMap, key)
	}
}

// AgentBroadcast broadcasts the notification. If no Agent Ctrls are currently
// connected notifications are buffered and agency send them immediately any of
// the controllers connect.
//
// TODO: add persistence that agency can be restarted.
func (m mapIndex) AgentBroadcast(state AgentNotify) {
	AgentMaps[m].Lock()
	defer AgentMaps[m].Unlock()

	if !m.broadcast(&state) { //
		glog.V(3).Infoln(state.ClientID, "there are no one to listen us!")
		if m == WantAllAgentActions {
			m.pushBufferedNotify(&state)
		}
		return
	}
}

func (m mapIndex) pushBufferedNotify(state *AgentNotify) {
	AgentMaps[m].Unlock()
	AgentMaps[m].buffer.Lock()
	defer AgentMaps[m].buffer.Unlock()
	defer AgentMaps[m].Lock()

	if AgentMaps[m].buffer.buf != nil {
		glog.V(13).Infoln(state.ClientID, "+++ push buffered", state.AgentDID)
		AgentMaps[m].buffer.buf.PushBack(state)
	}
}

// lockedBroadcast is same as broadcast function but thread save version of it.
// It is specifically made for algorithms which requires this locking order.
// It frees buffer level locks first and acquires map level lock before
// forecast. In the end it does it in reserve order. Go's normal defer is not
// used to make easier to read what functions does.
func (m mapIndex) lockedBroadcast(state *AgentNotify) (sent bool) {
	AgentMaps[m].buffer.Unlock() // Free buffer lock which was on in caller
	AgentMaps[m].Lock()          // Lock map level lock for broadcast function

	sent = m.broadcast(state)

	AgentMaps[m].Unlock()      // first free the map level lock
	AgentMaps[m].buffer.Lock() // 2nd put our lock for buffer on

	return sent
}

// broadcast broadcasts notification to listeners. Note! It doesn't lock
// the maps.
func (m mapIndex) broadcast(state *AgentNotify) (found bool) {
	broadcastKey := state.AgentKeyType
	for listenKey, ch := range AgentMaps[m].agentStationMap {
		hit := broadcastKey.AgentDID == listenKey.AgentDID ||
			listenKey.AgentDID == AllAgents
		if hit {
			found = hit
			glog.V(3).Infoln(broadcastKey.AgentDID,
				"agent broadcast notify: ", listenKey.ClientID)
			sendState := *state
			sendState.ClientID = listenKey.ClientID
			ch <- sendState
		}
	}
	return found
}
