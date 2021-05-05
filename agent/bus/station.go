package bus

import (
	"sync"

	"github.com/findy-network/findy-agent/agent/psm"
)

var ReadyStation = New()

func New() *Station {
	return &Station{channels: make(map[KeyType]Ready)}
}

type KeyType = psm.StateKey
type Ready chan bool
type StateChan chan psm.SubState

func newReady() Ready {
	return make(Ready, 1) // We need a buffered channel
}

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

func (s *Station) BroadcastReady(key KeyType, ok bool) {
	s.lk.Lock()

	c, found := s.channels[key]
	if !found {
		s.lk.Unlock()
		return
	}
	// we broadcast the ready-info only once
	delete(s.channels, key)
	s.lk.Unlock()

	c <- ok
}

func (s *Station) StartListen(key KeyType) <-chan bool {
	s.lk.Lock()
	defer s.lk.Unlock()

	c := newReady()
	s.channels[key] = c
	return c
}
