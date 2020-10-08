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

type StationMap map[KeyType]StateChan

var SubState = struct {
	StationMap
	sync.Mutex
}{StationMap: make(StationMap)}

func AddListener(key KeyType) StateChan {
	SubState.Lock()
	defer SubState.Unlock()

	SubState.StationMap[key] = make(StateChan)
	return SubState.StationMap[key]
}

func RmListener(key KeyType) {
	SubState.Lock()
	defer SubState.Unlock()
	delete(SubState.StationMap, key)
}

func Broadcast(key KeyType, state psm.SubState) {
	SubState.Lock()
	defer SubState.Unlock()

	c, ok := SubState.StationMap[key]
	if !ok {
		return
	}

	c <- state
}

func (s *Station) BroadcastReady(key KeyType, ok bool) {
	s.lk.Lock()
	defer s.lk.Unlock()

	c, found := s.channels[key]
	if !found {
		return
	}

	// we broadcast the ready-info only once
	delete(s.channels, key)
	c <- ok
}

func (s *Station) StartListen(key KeyType) <-chan bool {
	s.lk.Lock()
	defer s.lk.Unlock()

	c := newReady()
	s.channels[key] = c
	return c
}
