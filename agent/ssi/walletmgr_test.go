package ssi

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/utils"

	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

const (
	key         = "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"
	walletName1 = "wallet_mgr_wallet1"
	walletName2 = "wallet_mgr_wallet2"
	walletName3 = "wallet_mgr_wallet3"
)

func TestMain(m *testing.M) {
	pt := err2.PanicTracer()
	err2.SetPanicTracer(os.Stderr)

	setUp()
	code := m.Run()
	tearDown()

	err2.SetErrorTracer(pt)
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
	assert.PushTester(t)
	defer assert.PopTester()

	tests := []struct {
		name  string
		count int
	}{
		{"open size 1", 1},
		{"open size 2", 2},
		{"open size 3", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()

			SetWalletMgrPoolSize(tt.count)

			cfg := NewRawWalletCfg(walletName1, key)
			w := wallets.Open(cfg)
			glog.V(3).Info("read handle 1")
			assert.That(w.Handle() > 0)

			w.Handle()
			w.Handle()
			w.Handle()
			w.Handle()

			cfg = NewRawWalletCfg(walletName2, key)
			time.Sleep(time.Nanosecond) // 'real' work for underlying algorithm

			w2 := wallets.Open(cfg)
			glog.V(3).Info("read handle 2")
			assert.That(w2.Handle() > 0)

			w2.Handle()
			w2.Handle()
			w2.Handle()

			glog.V(3).Info("read handle 1")
			time.Sleep(time.Nanosecond) // 'real' work for underlying algorithm

			assert.That(w.Handle() > 0)
			w.Handle()
			w.Handle()
			w.Handle()

			cfg = NewRawWalletCfg(walletName3, key)
			time.Sleep(time.Nanosecond) // 'real' work for underlying algorithm
			w3 := wallets.Open(cfg)
			glog.V(3).Info("read handle 3")
			assert.That(w3.Handle() > 0)

			glog.V(3).Info("read handle 2")
			assert.That(w2.Handle() > 0)
			w2.Handle()
			w2.Handle()

			wallets.Reset()
		})
	}
}
