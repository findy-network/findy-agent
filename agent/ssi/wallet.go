package ssi

import (
	"os"
	"path/filepath"

	"github.com/findy-network/findy-agent/agent/utils"

	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/golang/glog"
)

type Wallet struct {
	Config      wallet.Config
	Credentials wallet.Credentials
	worker      bool
}

const WalletAlreadyExistsError = 203

func NewWalletCfg(name, key string) (w *Wallet) {
	return &Wallet{
		Config: wallet.Config{ID: name},
		Credentials: wallet.Credentials{
			Key:                 key,
			KeyDerivationMethod: "ARGON2I_MOD",
		},
	}
}

func NewRawWalletCfg(name, key string) (w *Wallet) {
	return &Wallet{
		Config: wallet.Config{ID: name},
		Credentials: wallet.Credentials{
			Key:                 key,
			KeyDerivationMethod: "RAW",
		},
	}
}

// WorkerWallet makes a copy of the wallet cfg, normally CA`s wallet
func (w Wallet) WorkerWallet() *Wallet {
	const suffix = "_w"
	return w.WorkerWalletBy(suffix)
}

// WorkerWalletBy makes a copy of the wallet cfg which name ends with suffix
func (w Wallet) WorkerWalletBy(suffix string) *Wallet {
	walletPath := workerWalletPath()
	w.Config.StorageConfig = &wallet.StorageConfig{Path: walletPath}
	w.Config.ID += suffix
	w.worker = true
	return &w
}

func workerWalletPath() string {
	const workerSubPath = "/.indy_client/worker"

	home := utils.IndyBaseDir()
	return filepath.Join(home, workerSubPath)
}

func walletPath() string {
	const workerSubPath = "/.indy_client/wallet"

	home := utils.IndyBaseDir()
	return filepath.Join(home, workerSubPath)
}

func (w *Wallet) Create() (exist bool) {
	r := <-wallet.Create(w.Config, w.Credentials)
	if r.Err() != nil {
		//	already exist, not real error, let it thru
		if WalletAlreadyExistsError != r.ErrCode() {
			panic(r.Err()) // panic with error type, err2 will catch
		}
		return true
	}
	return false
}

func (w *Wallet) SyncOpen() int {
	return w.Open().Int()
}

func (w *Wallet) Open() (f *Future) {
	if glog.V(3) {
		glog.Info("opening wallet: ", w.Config.ID)
	}
	f = new(Future)
	f.SetChan(wallet.Open(w.Config, w.Credentials))
	return f
}

func (w *Wallet) Exists(worker bool) bool {
	path := walletPath()
	if worker {
		path = workerWalletPath()
	}
	name := filepath.Join(path, w.Config.ID)
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func (w *Wallet) UniqueID() string {
	path := walletPath()
	if w.worker {
		path = workerWalletPath()
	}
	return filepath.Join(path, w.Config.ID)
}

func (w *Wallet) SyncClose(handle int) (err error) {
	return w.Close(handle).Result().Err()
}

func (w *Wallet) Close(handle int) (f *Future) {
	if glog.V(3) {
		glog.Infof("closing wallet(%d): %s", handle, w.Config.ID)
	}
	f = new(Future)
	f.SetChan(wallet.Close(handle))
	return f
}

func (w *Wallet) SetID(id string) {
	w.Config.ID = id
}

func (w *Wallet) SetKey(key string) {
	w.Credentials.Key = key
}

func (w *Wallet) SetKeyMethod(m string) {
	w.Credentials.KeyDerivationMethod = m
}

func (w *Wallet) ID() string {
	return w.Config.ID
}

func (w *Wallet) Key() string {
	return w.Credentials.Key
}
