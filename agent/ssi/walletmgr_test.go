package ssi

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/utils"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

const (
	key         = "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"
	walletName1 = "wallet_mgr_wallet1"
	walletName2 = "wallet_mgr_wallet2"
	walletName3 = "wallet_mgr_wallet3"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	home := utils.IndyBaseDir()
	removeFiles(home, "/.indy_client/wallet/wallet_mgr_wallet*")
}

func removeFiles(home, nameFilter string) {
	filter := filepath.Join(home, nameFilter)
	files, _ := filepath.Glob(filter)
	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			panic(err)
		}
	}
}

func setUp() {
	createTestWallets()
}

func createTestWallets() {
	wallet := NewRawWalletCfg(walletName1, key)
	wallet.Create()
	wallet = NewRawWalletCfg(walletName2, key)
	wallet.Create()
	wallet = NewRawWalletCfg(walletName3, key)
	wallet.Create()
}

func TestMgr_NewOpen(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"open size 1", 1},
		{"open size 2", 2},
		{"open size 3", 3},
	}
	for _, tt := range tests {
		SetWalletMgrPoolSize(tt.count)

		cfg := NewRawWalletCfg(walletName1, key)
		w := Wallets.Open(cfg)
		glog.V(3).Info("read handle 1")
		assert.Greater(t, w.Handle(), 0)

		w.Handle()
		w.Handle()
		w.Handle()
		w.Handle()

		cfg = NewRawWalletCfg(walletName2, key)
		time.Sleep(time.Nanosecond) // 'real' work for underlying algorithm

		w2 := Wallets.Open(cfg)
		glog.V(3).Info("read handle 2")
		assert.Greater(t, w2.Handle(), 0)

		w2.Handle()
		w2.Handle()
		w2.Handle()

		glog.V(3).Info("read handle 1")
		time.Sleep(time.Nanosecond) // 'real' work for underlying algorithm

		assert.Greater(t, w.Handle(), 0)
		w.Handle()
		w.Handle()
		w.Handle()

		cfg = NewRawWalletCfg(walletName3, key)
		time.Sleep(time.Nanosecond) // 'real' work for underlying algorithm
		w3 := Wallets.Open(cfg)
		glog.V(3).Info("read handle 3")
		assert.Greater(t, w3.Handle(), 0)

		glog.V(3).Info("read handle 2")
		assert.Greater(t, w2.Handle(), 0)
		w2.Handle()
		w2.Handle()

		Wallets.Reset()
	}
}
