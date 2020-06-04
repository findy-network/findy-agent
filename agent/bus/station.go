package bus

import (
	"sync"

	"github.com/optechlab/findy-agent/agent/psm"
)

var ReadyStation = New()

func New() *Station {
	return &Station{channels: make(map[KeyType]Ready)}
}

type KeyType = psm.StateKey
type Ready chan bool

func newReady() Ready {
	return make(Ready, 1) // We need a buffered channel
}

type Station struct {
	channels map[KeyType]Ready
	lk       sync.Mutex
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
