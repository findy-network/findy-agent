package ssi

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/findy-network/findy-agent/agent/accessmgr"
	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/storage/api"
	storage "github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

var maxOpened = 10

// SetWalletMgrPoolSize sets pool size, i.e. how many wallets can kept open in
// the same time. This should be set at the startup of the application or
// service.
func SetWalletMgrPoolSize(s int) {
	wallets.l.Lock() // Precaution
	defer wallets.l.Unlock()

	maxOpened = s
}

// Handle implements ManagedWallet interface. These types together offer an API
// to use SSI wallets conveniently. They hide closing and opening logic which is
// needed to reserve OS level file handles. Only limited amount of simultaneous
// wallet handles is kept open (MaxOpen). See more information from API function
// descriptions.
type Handle struct {
	ts      int64                // last access timestamp
	h       int                  // wallet handle
	f       *Future              // wallet handle future
	cfg     Wallet               // wallet file information
	storage storage.AgentStorage // agent-specific storage
	l       sync.RWMutex         // lock
}

// Config returns managed wallet's associated indy wallet configuration.
func (h *Handle) Config() managed.WalletCfg {
	h.l.RLock()
	defer h.l.RUnlock()
	return &h.cfg
}

// Close frees the wallet handle to reuse by WalletMgr. Please note that it's
// NOT important or desired to call this function during the agency process is
// running.
func (h *Handle) Close() {
	defer err2.Catch(func(err error) {
		glog.Warning("closing error:", err)
	})

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
	err2.Check(f.Result().Err())
	err2.Check(h.storage.Close())
}

func (h *Handle) timestamp() int64 {
	h.l.RLock()
	defer h.l.RUnlock()
	return h.ts
}

func (h *Handle) Storage() api.AgentStorage {
	return h.storage
}

// Handle returns the actual indy wallet handle which can be used with indy SDK
// API calls. The Handle function hides all the needed complexity behind it. For
// example, if the actual libindy wallet handle is already closed, it will be
// opened first. Please note that there is no performance penalty i.e. no
// optimization is needed.
func (h *Handle) Handle() int {
	h.l.Lock()
	defer h.l.Unlock()

	if handle := h.h; handle != 0 {
		h.ts = time.Now().UnixNano()
		return handle
	}

	// reopen with the Manager. Note! They know that handle is locked
	return wallets.reopen(h)
}

// reopen opens the wallet by its configuration. Open is always called by Wallet
// Manager because it will keep track of wallet handles and max amount of them.
func (h *Handle) reopen() int {
	defer err2.Catch(func(err error) {
		glog.Error("error when reopening wallet: ", err)
	})

	h.f = h.cfg.Open()
	if glog.V(10) {
		glog.Info("opening wallet: ", h.cfg.Config.ID)
	}
	h.ts = time.Now().UnixNano()
	h.h = h.f.Int()

	err2.Check(h.storage.Open())

	return h.h
}

type WalletMap map[string]*Handle

type Mgr struct {
	opened WalletMap
	l      sync.Mutex // lock
}

var wallets = &Mgr{
	opened: make(WalletMap, maxOpened),
}

// Open opens a wallet configuration and returns a managed wallet.
func (m *Mgr) Open(cfg Wallet) managed.Wallet {
	m.l.Lock()
	defer m.l.Unlock()

	if len(m.opened) < maxOpened {
		return m.openNewWallet(cfg)
	}

	// we have exceeded max opened count, move the oldest to closed ones
	return m.closeOldestAndOpen(cfg)
}

func (m *Mgr) openNewWallet(cfg Wallet) managed.Wallet {
	defer err2.Catch(func(err error) {
		glog.Error("error when opening wallet: ", err)
	})

	home := utils.IndyBaseDir()
	filePath := filepath.Join(home, ".indy_client") // TODO: fetch from settings

	aStorage, err := mgddb.New(storage.AgentStorageConfig{
		AgentID:  cfg.ID(),
		AgentKey: mgddb.GenerateKey(), // TODO: fetch from settings
		FilePath: filePath,
	})
	err2.Check(err)

	h := &Handle{
		ts:      time.Now().UnixNano(),
		h:       0,
		f:       cfg.Open(),
		cfg:     cfg,
		storage: aStorage,
	}
	m.opened[cfg.UniqueID()] = h
	h.h = h.f.Int()

	if h.cfg.worker {
		// AccessMgr will handle backups. Let it know that the managed WORKER
		// wallet is opened. Pairwise wallet backup will be handled in
		// handshake.
		accessmgr.Send(h)
	}

	return h
}

func (m *Mgr) reopen(h *Handle) int {
	m.l.Lock()
	defer m.l.Unlock()

	if len(m.opened) < maxOpened {
		m.opened[h.cfg.UniqueID()] = h
		return h.reopen()
	}

	return m.closeOldestAndReopen(h)
}

func (m *Mgr) closeOldestAndReopen(h *Handle) int {
	oldest := m.findOldest()
	w := m.opened[oldest]
	w.Close()
	delete(m.opened, oldest)
	m.opened[h.cfg.UniqueID()] = h

	return h.reopen()
}

func (m *Mgr) closeOldestAndOpen(cfg Wallet) managed.Wallet {
	oldest := m.findOldest()
	w := m.opened[oldest]
	delete(m.opened, oldest)
	w.Close()

	return m.openNewWallet(cfg)
}

func (m *Mgr) findOldest() string {
	var id string
	var maxDelta int64
	now := time.Now().UnixNano()

	for s, wallet := range m.opened {
		delta := now - wallet.timestamp()
		if delta > maxDelta {
			maxDelta = delta
			id = s
		}
	}
	return id
}

// Reset resets the managed wallet buffer which means that all the current
// wallet configurations must be registered again with ssi.Wallets.Open. Note!
// You should not need to use this!
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
