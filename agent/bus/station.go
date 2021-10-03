package bus

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/psm"
)

type KeyType = psm.StateKey
type Ready chan bool
type StateChan chan psm.SubState

type Station struct {
	channels map[KeyType]Ready
	lk       sync.Mutex
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

var Maps = [...]lockMap{{stationMap: make(stationMap)}, {stationMap: make(stationMap)}}
var WantAll mapIndex = allStates
var WantUserActions mapIndex = userActions

func (m mapIndex) AddListener(key KeyType) StateChan {
	Maps[m].Lock()
	defer Maps[m].Unlock()

	Maps[m].stationMap[key] = make(StateChan)
	return Maps[m].stationMap[key]
}

func (m mapIndex) RmListener(key KeyType) {
	Maps[m].Lock()
	defer Maps[m].Unlock()
	delete(Maps[m].stationMap, key)
}

func (m mapIndex) Broadcast(key KeyType, state psm.SubState) {
	// We need to unlock as soon as possible that we don't keep lock when
	// blocking channel send at the end.
	Maps[m].Lock()

	c, ok := Maps[m].stationMap[key]
	if !ok {
		Maps[m].Unlock() // Manual unlock needed, see below
		return
	}
	Maps[m].Unlock() // Important! Leve lock before writing channel

	c <- state
}
