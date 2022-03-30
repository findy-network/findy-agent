package cfg

import (
	"path/filepath"
	"sync"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type AgentStorage struct {
	api.AgentStorageConfig
}

type StorageInfo struct {
	storage *mgddb.Storage
	handle  int
	isOpen  bool
}

type InfoMap map[string]StorageInfo

var (
	storages = struct {
		InfoMap
		handles []*mgddb.Storage
		sync.RWMutex
	}{
		InfoMap: make(InfoMap),
		handles: make([]*mgddb.Storage, 0, 12),
	}
)

func Storage(h int) api.AgentStorage {
	storages.RLock()
	defer storages.RUnlock()

	s := storages.handles[h-1]
	glog.V(11).Infof("----- STORAGE: (%s) HANDLE: (%d)", s.StorageProvider.ID(), h)
	return storages.handles[h-1] // handles start from 1 not 0
}

func (c *AgentStorage) UniqueID() string {
	return filepath.Join(c.FilePath, c.AgentID)
}

func (c *AgentStorage) ID() string {
	return c.AgentID
}

func (c *AgentStorage) Key() string {
	return c.AgentKey
}

func (c *AgentStorage) OpenWallet() (h int, err error) {
	defer err2.Annotate("open agent storage from cfg", &err)

	storages.Lock()
	defer storages.Unlock()

	info, exist := storages.InfoMap[c.UniqueID()]
	if exist {
		try.To(info.storage.Open())
		glog.V(5).Infoln("open existing agent storage:", c.AgentID)
		info.isOpen = true
		storages.InfoMap[c.UniqueID()] = info
		return info.handle, nil
	}

	// reserve zero for nul handle, start handles from 1
	handle := len(storages.InfoMap) + 1

	aStorage := try.To1(mgddb.New(c.AgentStorageConfig))
	try.To(aStorage.Open())
	glog.V(5).Infoln("successful first time opening agent storage:", c.AgentID)

	storages.InfoMap[c.UniqueID()] = StorageInfo{
		storage: aStorage,
		handle:  handle,
		isOpen:  true,
	}
	storages.handles = append(storages.handles, aStorage)
	assert.That(len(storages.handles) == len(storages.InfoMap), "fast index mismatch")

	return handle, nil
}

func (c *AgentStorage) CloseWallet(handle int) (err error) {
	defer err2.Annotate("close agent storage from cfg", &err)

	storages.Lock()
	defer storages.Unlock()

	info, exist := storages.InfoMap[c.UniqueID()]
	assert.That(exist, "config must exists in map")
	assert.That(info.handle == handle, "info.handle must be equal to argument handle")

	if info.isOpen {
		try.To(info.storage.Close())
		glog.V(5).Infoln("successful closing agent storage:", c.AgentID)
		// closing flag is updated only if Close() success
		info.isOpen = false
		storages.InfoMap[c.UniqueID()] = info
	} else {
		glog.Warningf("CloseWallet called but wallet (%s) not open!",
			c.UniqueID())
	}
	return nil
}

func (c *AgentStorage) WantsBackup() bool {
	return true
}
