package ssi

import (
	"sync"
	"time"

	"github.com/golang/glog"
)

var maxOpened = 100

type Handle struct {
	ts  int64        // last access timestamp
	h   int          // wallet handle
	f   *Future      // wallet handle future
	cfg *Wallet      // wallet file information
	l   sync.RWMutex // lock
}

func (h *Handle) Config() *Wallet {
	h.l.RLock()
	defer h.l.RUnlock()
	return h.cfg
}

func (h *Handle) Close() {
	h.l.Lock()
	defer h.l.Unlock()
	f := h.cfg.Close(h.f.Int())
	if glog.V(10) {
		glog.Info("closing wallet: ", h.cfg.Config.ID)
	}
	if h.h != h.f.Int() {
		glog.Warning("handle mismatch!!")
	}
	h.h = 0
	if f.Result().Err() != nil {
		glog.Warning("closing error:", f.Result().Err())
	}
}

func (h *Handle) timestamp() int64 {
	h.l.RLock()
	defer h.l.RUnlock()
	return h.ts
}

func (h *Handle) Handle() int {
	h.l.Lock()
	if handle := h.h; handle != 0 {
		h.ts = time.Now().UnixNano()
		h.l.Unlock()
		return handle
	}
	h.l.Unlock()

	// reopen with the Manager
	return Wallets.reopen(h)
}

func (h *Handle) Open() int {
	h.l.Lock()
	defer h.l.Unlock()
	h.f = h.cfg.Open()
	if glog.V(10) {
		glog.Info("opening wallet: ", h.cfg.Config.ID)
	}
	h.ts = time.Now().UnixNano()
	h.h = h.f.Int()
	return h.h
}

type ManagedWallet interface {
	Open() int
	Close()
	Handle() int
	Config() *Wallet
}

type WalletMap map[string]*Handle

type Mgr struct {
	opened WalletMap
	l      sync.RWMutex // lock
}

var Wallets = &Mgr{
	opened: make(WalletMap, maxOpened),
}

func (m *Mgr) Open(cfg *Wallet) ManagedWallet {
	m.l.RLock()
	if len(m.opened) < maxOpened {
		m.l.RUnlock()
		return m.openNewWallet(cfg)
	}
	m.l.RUnlock()

	// we have exceeded max opened count, move the oldest of the opened once to
	// closed one
	return m.closeOldestAndOpen(cfg)
}

func (m *Mgr) openNewWallet(cfg *Wallet) ManagedWallet {
	h := &Handle{
		ts:  time.Now().UnixNano(),
		h:   0,
		f:   cfg.Open(),
		cfg: cfg,
	}
	m.l.Lock()
	defer m.l.Unlock()
	m.opened[cfg.UniqueID()] = h
	h.h = h.f.Int()
	return h
}

func (m *Mgr) reopen(h *Handle) int {
	m.l.Lock()
	if len(m.opened) < maxOpened {
		m.opened[h.cfg.UniqueID()] = h
		m.l.Unlock()
		return h.Open()
	}
	m.l.Unlock()

	return m.closeOldestAndReopen(h)
}

func (m *Mgr) closeOldestAndReopen(h *Handle) int {
	oldest := m.findOldest()

	m.l.Lock()
	defer m.l.Unlock()
	w := m.opened[oldest]
	w.Close()
	delete(m.opened, oldest)
	m.opened[h.cfg.UniqueID()] = h

	return h.Open()
}

func (m *Mgr) closeOldestAndOpen(cfg *Wallet) ManagedWallet {
	oldest := m.findOldest()

	m.l.Lock()
	w := m.opened[oldest]
	delete(m.opened, oldest)
	w.Close()
	m.l.Unlock()

	return m.openNewWallet(cfg)
}

func (m *Mgr) findOldest() string {
	var id string
	var maxDelta int64
	now := time.Now().UnixNano()

	m.l.RLock()
	defer m.l.RUnlock()
	for s, wallet := range m.opened {
		delta := now - wallet.timestamp()
		if delta > maxDelta {
			maxDelta = delta
			id = s
		}
	}
	return id
}

func (m *Mgr) Reset() {
	if glog.V(3) {
		glog.Infof("resetting %d wallets", len(m.opened))
	}
	m.l.Lock()
	defer m.l.Unlock()
	for _, wallet := range m.opened {
		wallet.Close()
	}
	m.opened = make(WalletMap, maxOpened)
}
