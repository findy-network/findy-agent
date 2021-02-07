package accessmgr

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

type chanType chan managed.Wallet
type mapType map[string]managed.Wallet

var (
	input    = make(chanType, 10) // make short performance buffer
	accessed = struct {
		Map mapType
		sync.Mutex
	}{Map: make(mapType)}

	DateTimeInName = true
	enabled        = utils.Settings.WalletBackupPath() != ""
	started        = false
)

// Send sends the managed wallet to input channel if accessmgr is enabled. It
// also returns the current Enable status.
func Send(mw managed.Wallet) bool {
	if enabled {
		assert.D.True(started, "access manager must be started!")

		input <- mw
	}
	return enabled
}

// Start starts the Access Mgr for the managed wallets if it's enabled. Access
// Mgr is enabled if WalletBackupPath agency settings is set.
func Start() {
	assert.D.True(enabled, "wallet backup path must be set!")

	started = true
	go func() {
		defer err2.CatchTrace(func(err error) {
			glog.Error(err)
		})
		glog.V(1).Infoln("wallet access mgr started")
		for walletCfg := range input {
			accessed.Lock()

			_, ok := accessed.Map[walletCfg.Config().UniqueID()]
			if ok {
				glog.V(1).Infoln("wallet access already registered")
			}
			accessed.Map[walletCfg.Config().UniqueID()] = walletCfg
			accessed.Unlock()
		}
	}()
}

// StartBackup starts the backup process for the managed wallets. Access Mgr is
// enabled if WalletBackupPath agency settings is set.
func StartBackup() {
	if !enabled {
		glog.Warning("wallet backup disabled")
		return
	}

	accessed.Lock()
	defer accessed.Unlock()

	newMap := accessed.Map
	accessed.Map = make(mapType)

	go runBackup(newMap)
}

func runBackup(m mapType) {
	for id, managedWallet := range m {
		if err := backup(managedWallet); err != nil {
			glog.Error("error in backup:", err)
		} else {
			glog.V(1).Infoln("successful wallet backup:", id)
		}
	}
}

func backup(mw managed.Wallet) (err error) {
	cfg := mw.Config()
	exportCredentials := BuildExportCredentials(cfg)
	r := <-wallet.Export(mw.Handle(), exportCredentials)
	return r.Err()
}

func BuildExportCredentials(cfg managed.WalletCfg) wallet.Credentials {
	exportFile := utils.Settings.WalletBackupPath()
	exportFile = filepath.Join(exportFile, backupName(cfg.ID()))
	exportCreds := wallet.Credentials{
		Path:                exportFile,
		Key:                 cfg.Key(),
		KeyDerivationMethod: "RAW",
	}
	return exportCreds
}

func backupName(baseName string) string {
	if !DateTimeInName {
		return baseName
	}
	tsStr := time.Now().Format(time.RFC3339)
	name := tsStr + "_" + baseName
	glog.V(3).Infoln("backup name:", name)
	return name
}
