package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/lainio/err2"
)

const (
	walletName1  = "test_wallet1"
	walletName2  = "test_wallet2"
	walletKey    = "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"
	email2       = "test_email2"
	testGenesis  = "../configs/test/genesis_tranactions"
	importWallet = "../configs/test/importWallet"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	currentUser, err := user.Current()
	err2.Check(err)
	home := currentUser.HomeDir

	removeFiles(home, "/.indy_client/worker/test_wallet*")
	removeFiles(home, "/.indy_client/worker/test_email*")
	removeFiles(home, "/.indy_client/wallet/test_*")
	removeFiles(home, "/.indy_client/wallet/test_email*")
	removeFiles(home, "/test_export_wallets/*")
	removeFile(testGenesis)
	removeFile(importWallet)
	ssi.ClosePool()
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
func removeFile(filename string) {
	if err := os.Remove(filename); err != nil {
		panic(err)
	}

}

func setUp() {
	defer err2.CatchTrace(func(err error) {
		fmt.Println("error on setup", err)
	})
	err2.Try(createTestWallets())
	f, e := os.Create(testGenesis)
	err2.Check(e)
	defer f.Close()
	impFile, e2 := os.Create(importWallet)
	err2.Check(e2)
	defer impFile.Close()
}

func createTestWallets() (err error) {
	wallet1 := ssi.NewRawWalletCfg(walletName1, walletKey)
	exist := wallet1.Create()
	if exist {
		return errors.New("test wallet exist already")
	}
	return nil
}

func TestExecute(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Define tests
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "tools create key",
			args: []string{"cmd",
				"tools", "key", "create", "--dry-run",
				"--seed", "00000000000000000000thisisa_test",
			},
		},
		{
			name: "user ping",
			args: []string{"cmd",
				"user", "ping", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
			},
		},
		{
			name: "service ping",
			args: []string{"cmd",
				"service", "ping", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
			},
		},
		{
			name: "user send basic msg",
			args: []string{"cmd",
				"user", "send", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--msg", "test message",
				"--from", "senderName",
				"--connection-id", "connectionID",
			},
		},
		{
			name: "service send basic msg",
			args: []string{"cmd",
				"service", "send", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--msg", "test message",
				"--from", "senderName",
				"--connection-id", "connectionID",
			},
		},
		{
			name: "user trustping",
			args: []string{"cmd",
				"user", "trustping", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--connection-id", "my_connectionID",
			},
		},
		{
			name: "service trustping",
			args: []string{"cmd",
				"service", "trustping", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--connection-id", "my_connectionID",
			},
		},
		{
			name: "user onboard",
			args: []string{"cmd",
				"user", "onboard", "--dry-run",
				"--email", email2,
				"--wallet-name", walletName2,
				"--wallet-key", walletKey,
			},
		},
		{
			name: "service onboard",
			args: []string{"cmd",
				"service", "onboard", "--dry-run",
				"--email", email2,
				"--wallet-name", walletName2,
				"--wallet-key", walletKey,
			},
		},
		{
			name: "create schema (config file)",
			args: []string{"cmd",
				"service", "schema", "create", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--config", "../configs/test/createSchema.yaml",
			},
		},
		{
			name: "service read schema",
			args: []string{"cmd",
				"service", "schema", "read", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--id", "my_schema_id",
			},
		},
		{
			name: "user read schema",
			args: []string{"cmd",
				"user", "schema", "read", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--id", "my_schema_id",
			},
		},
		{
			name: "create creddef (config file)",
			args: []string{"cmd",
				"service", "creddef", "create", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--config", "../configs/test/createCreddef.yaml",
			},
		},
		{
			name: "service read creddef",
			args: []string{"cmd",
				"service", "creddef", "read", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--id", "my_creddef_id",
			},
		},
		{
			name: "user read creddef",
			args: []string{"cmd",
				"user", "creddef", "read", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--id", "my_creddef_id",
			},
		},
		{
			name: "create steward",
			args: []string{"cmd",
				"ledger", "steward", "create", "--dry-run",
				"--pool-name", "test-pool",
				"--seed", "000000000000000000000000Steward4",
				"--wallet-name", "steward-wallet",
				"--wallet-key", walletKey,
			},
		},
		{
			name: "create pool",
			args: []string{"cmd",
				"ledger", "pool", "create", "--dry-run",
				"--name", "findy-pool",
				"--genesis-txn-file", testGenesis,
			},
		},
		{
			name: "ping pool",
			args: []string{"cmd",
				"ledger", "pool", "ping", "--dry-run",
				"--name", "findy-pool",
			},
		},
		{
			name: "start agency (config file)",
			args: []string{"cmd",
				"agency", "start", "--dry-run",
				"--config", "../configs/test/startAgency.yaml",
			},
		},
		{
			name: "ping agency",
			args: []string{"cmd",
				"agency", "ping", "--dry-run",
				"--base-address", "my_agency_base_address.com",
			},
		},
		{
			name: "user create invitation",
			args: []string{"cmd",
				"user", "invitation", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--label", "connection_name",
			},
		},
		{
			name: "service create invitation",
			args: []string{"cmd",
				"service", "invitation", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--label", "connection_name",
			},
		},
		{
			name: "service connect (config file & no invitation)",
			args: []string{"cmd",
				"service", "connect", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--config", "../configs/test/connect.yaml",
			},
		},
		{
			name: "user connect (config file & no invitation)",
			args: []string{"cmd",
				"user", "connect", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--config", "../configs/test/connect.yaml",
			},
		},
		{
			name: "tools wallet export",
			args: []string{"cmd",
				"tools", "export", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"--key", walletKey,
				"--file", "../configs/test/my-export-wallet",
			},
		},
		{
			name: "tools wallet import",
			args: []string{"cmd",
				"tools", "import", "--dry-run",
				"--wallet-name", "testWallet",
				"--wallet-key", walletKey,
				"--key", walletKey,
				"--file", importWallet,
			},
		},
		{
			name: "service connect invitation",
			args: []string{"cmd",
				"service", "connect", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"../configs/test/test_invitation",
			},
		},
		{
			name: "user connect invitation",
			args: []string{"cmd",
				"user", "connect", "--dry-run",
				"--wallet-name", walletName1,
				"--wallet-key", walletKey,
				"../configs/test/test_invitation",
			},
		},
	}

	// Iterate tests
	for _, test := range tests {
		os.Args = test.args
		rootCmd.SilenceUsage = true
		rootCmd.SilenceErrors = true

		t.Run(test.name, func(t *testing.T) {
			if err := rootCmd.Execute(); err != nil {
				t.Errorf("Test error = %v", err)
			}
		})
	}
}
