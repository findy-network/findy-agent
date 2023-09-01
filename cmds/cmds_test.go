package cmds_test

import (
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/findy-network/findy-agent/agent/pool"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/agency"
	stewardCmd "github.com/findy-network/findy-agent/cmds/steward"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/findy-network/findy-agent/server"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

const (
	stewardTmpWalletName1 = "unit_test_steward_wallet1"
	stewardTmpWalletKey1  = "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"
)

var (
	agencyCmd      agency.Cmd
	httpTestServer *httptest.Server
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	httpTestServer.Close()

	home := utils.IndyBaseDir()

	removeFiles(home, "/.indy_client/worker/unit_test_wallet*")
	removeFiles(home, "/.indy_client/worker/email*")
	removeFiles(home, "/.indy_client/wallet/unit_test_*")
	removeFiles(home, "/.indy_client/wallet/email*")
	removeFiles(home, "/.indy_client/storage/unit_test_wallet*")
	removeFiles(home, "/.indy_client/storage/email*")
	removeFiles(home, "/export_wallets/*")
	enclave.WipeSealedBox()
	pool.Close()
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
	defer err2.Catch(err2.Err(func(err error) {
		fmt.Println("error on setup", err)
	}))

	enclavePath, exportPath := setupPaths()
	agencyCmd = agency.Cmd{
		PoolProtocol:      2,
		PoolName:          "FINDY_MEM_LEDGER",
		WalletName:        "sovrin_steward_wallet",
		WalletPwd:         "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE",
		StewardSeed:       "",
		ServiceName:       "a2a",
		HostAddr:          "localhost",
		HostPort:          8080,
		ServerPort:        0,
		EnclavePath:       enclavePath,
		ExportPath:        exportPath,
		StewardDid:        "Th7MpTaRZVRYnPiabds81Y",
		HandshakeRegister: "findy.json",
		PsmDB:             "findy.bolt",
		ResetData:         true, // IMPORTANT for testing!
		VersionInfo:       "test test",
	}
	try.To(agencyCmd.Validate())

	// We don't want logs on file with tests
	try.To(flag.Set("logtostderr", "true"))

	agencyCmd.PreRun()

	try.To(agencyCmd.Setup())

	httpTestServer = server.StartTestHTTPServer2()

	// note! We cannot call agencyCmd.CloseAll() because previous function does
	// not block. The tearDown() cleans up and closes all.
}

func setupPaths() (string, string) {
	exportPath := os.Getenv("TEST_WORKDIR")
	var sealedBoxPath string
	if len(exportPath) == 0 {
		exportPath = utils.IndyBaseDir()
		sealedBoxPath = filepath.Join(exportPath, ".indy_client/wallet/enclave.bolt")
	} else {
		sealedBoxPath = "enclave.bolt"
	}
	exportPath = filepath.Join(exportPath, "export_wallets")

	if os.Getenv("CI") == "true" {
		sw := ssi.NewRawWalletCfg("sovrin_steward_wallet", "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
		server.ResetEnv(sw, exportPath)
	}

	return sealedBoxPath, exportPath
}

func Test_CreateSteward(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	createStewardCmd := stewardCmd.CreateCmd{
		Cmd: cmds.Cmd{
			WalletName: stewardTmpWalletName1,
			WalletKey:  stewardTmpWalletKey1,
		},
		PoolName:    "FINDY_MEM_LEDGER",
		StewardSeed: "000000000000000000000000Steward2",
	}
	err := createStewardCmd.Validate()
	assert.NoError(err)
	_, err = createStewardCmd.Exec(os.Stdout)
	assert.NoError(err)
}

func Test_ValidateWalletExistence(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	cmd := cmds.Cmd{
		WalletName: stewardTmpWalletName1,
		WalletKey:  "",
	}
	err := cmd.ValidateWalletExistence(false)
	assert.Error(err)
	err = cmd.ValidateWalletExistence(true)
	assert.NoError(err)

	cmd = cmds.Cmd{
		WalletName: stewardTmpWalletName1 + "NOT_EXIST",
		WalletKey:  "",
	}
	err = cmd.ValidateWalletExistence(false)
	assert.NoError(err)
	err = cmd.ValidateWalletExistence(true)
	assert.Error(err)
}
