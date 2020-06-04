package cmds_test

import (
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/ssi"
	"github.com/optechlab/findy-agent/cmds"
	"github.com/optechlab/findy-agent/cmds/agency"
	"github.com/optechlab/findy-agent/cmds/agent"
	"github.com/optechlab/findy-agent/cmds/agent/creddef"
	"github.com/optechlab/findy-agent/cmds/agent/schema"
	"github.com/optechlab/findy-agent/cmds/connection"
	"github.com/optechlab/findy-agent/cmds/onboard"
	stewardCmd "github.com/optechlab/findy-agent/cmds/steward"
	"github.com/optechlab/findy-agent/enclave"
	"github.com/optechlab/findy-agent/server"
	didexchange "github.com/optechlab/findy-agent/std/didexchange/invitation"
	"github.com/stretchr/testify/assert"
)

const (
	stewardTmpWalletName1 = "unit_test_steward_wallet1"
	stewardTmpWalletKey1  = "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"

	walletName1 = "unit_test_wallet1"
	walletKey1  = "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"
	walletName2 = "unit_test_wallet2"
	walletKey2  = "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp"
	email1      = "email1"
	email2      = "email2"
)

var (
	agencyCmd        agency.Cmd
	invitation2      didexchange.Invitation
	httpTestServer   *httptest.Server
	walletExportPath string
	schemaID         string
	credDefID        string

	wallet1Cmd = cmds.Cmd{
		WalletName: walletName1,
		WalletKey:  walletKey1,
	}
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	httpTestServer.Close()

	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	home := currentUser.HomeDir

	removeFiles(home, "/.indy_client/worker/unit_test_wallet*")
	removeFiles(home, "/.indy_client/worker/email*")
	removeFiles(home, "/.indy_client/wallet/unit_test_*")
	removeFiles(home, "/.indy_client/wallet/email*")
	removeFiles(home, "/export_wallets/*")
	enclave.WipeSealedBox()
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

func setUp() {
	defer err2.CatchTrace(func(err error) {
		fmt.Println("error on setup", err)
	})

	enclavePath, exportPath := setupPaths()
	walletExportPath = exportPath
	agencyCmd = agency.Cmd{
		PoolProtocol:      2,
		PoolName:          "FINDY_MEM_LEDGER",
		PoolTxnName:       "",
		WalletName:        "sovrin_steward_wallet",
		WalletPwd:         "steward_wallet_key",
		StewardSeed:       "",
		ServiceName:       "ca-api",
		ServiceName2:      "a2a",
		HostAddr:          "localhost",
		HostPort:          8080,
		ServerPort:        0,
		EnclavePath:       enclavePath,
		ExportPath:        exportPath,
		StewardDid:        "Th7MpTaRZVRYnPiabds81Y",
		HandshakeRegister: "findy.json",
		PsmDb:             "findy.bolt",
		ResetData:         true, // IMPORTANT for testing!
		VersionInfo:       "test test",
		Salt:              "",
	}
	err2.Check(agencyCmd.Validate())

	// We don't want logs on file with tests
	err2.Check(flag.Set("logtostderr", "true"))

	agencyCmd.PreRun()

	err2.Check(agencyCmd.Setup())

	httpTestServer = server.StartTestHTTPServer2()

	// note! We cannot call agencyCmd.CloseAll() because previous function does
	// not block. The tearDown() cleans up and closes all.
}

func setupPaths() (string, string) {
	exportPath := os.Getenv("TEST_WORKDIR")
	var sealedBoxPath string
	if len(exportPath) == 0 {
		currentUser, _ := user.Current()
		exportPath = currentUser.HomeDir
		sealedBoxPath = filepath.Join(exportPath, ".indy_client/wallet/enclave.bolt")
	} else {
		sealedBoxPath = "enclave.bolt"
	}
	exportPath = filepath.Join(exportPath, "export_wallets")

	if os.Getenv("CI") == "true" {
		sw := ssi.NewWalletCfg("sovrin_steward_wallet", "steward_wallet_key")
		server.ResetEnv(sw, exportPath)
	}

	return sealedBoxPath, exportPath
}

func Test_CreatSteward(t *testing.T) {
	createStewardCmd := stewardCmd.CreateCmd{
		Cmd: cmds.Cmd{
			WalletName: stewardTmpWalletName1,
			WalletKey:  stewardTmpWalletKey1,
		},
		PoolName:    "FINDY_MEM_LEDGER",
		StewardSeed: "000000000000000000000000Steward2",
	}
	err := createStewardCmd.Validate()
	assert.NoError(t, err)
	_, err = createStewardCmd.Exec(os.Stdout)
	assert.NoError(t, err)
}

func Test_ValidateWalletExistence(t *testing.T) {
	cmd := cmds.Cmd{
		WalletName: stewardTmpWalletName1,
		WalletKey:  "",
	}
	err := cmd.ValidateWalletExistence(false)
	assert.Error(t, err)
	err = cmd.ValidateWalletExistence(true)
	assert.NoError(t, err)

	cmd = cmds.Cmd{
		WalletName: stewardTmpWalletName1 + "NOT_EXIST",
		WalletKey:  "",
	}
	err = cmd.ValidateWalletExistence(false)
	assert.NoError(t, err)
	err = cmd.ValidateWalletExistence(true)
	assert.Error(t, err)
}

func Test_AgencyPing(t *testing.T) {
	agencyPingCmd := agency.PingCmd{BaseAddr: httpTestServer.URL}
	err := agencyPingCmd.Validate()
	assert.NoError(t, err)
	_, err = agencyPingCmd.Exec(os.Stdout)
	assert.NoError(t, err)
}

func Test_Onboard(t *testing.T) {
	onboardCmd := onboard.Cmd{
		Cmd: cmds.Cmd{
			WalletName: walletName1,
			WalletKey:  walletKey1,
		},
		Email:      email1,
		AgencyAddr: httpTestServer.URL,
	}
	onboardCmd.Validate()
	_, err := onboardCmd.Exec(os.Stdout)
	assert.NoError(t, err)
	onboardCmd = onboard.Cmd{
		Cmd: cmds.Cmd{
			WalletName: walletName2,
			WalletKey:  walletKey2,
		},
		Email:      email2,
		AgencyAddr: httpTestServer.URL,
	}
	r2, err := onboardCmd.Exec(os.Stdout)
	assert.NoError(t, err)
	invitation2 = r2.Invitation
}

func Test_Export(t *testing.T) {
	exportPath := filepath.Join(walletExportPath, walletName1)
	exportPath = filepath.Join(exportPath, ".export")
	exportCmd := agent.ExportCmd{
		Cmd:       wallet1Cmd,
		Filename:  exportPath,
		ExportKey: walletKey1,
	}
	_, err := exportCmd.Exec(os.Stdout)
	assert.NoError(t, err)
}

func Test_Ping(t *testing.T) {
	cmd := agent.PingCmd{
		Cmd: wallet1Cmd,
	}
	_, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
}

func Test_Invite(t *testing.T) {
	cmd := agent.InvitationCmd{
		Cmd: wallet1Cmd,
	}
	_, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
}

func Test_ConnectionCmd(t *testing.T) {
	cmd := agent.ConnectionCmd{
		Cmd:        wallet1Cmd,
		Invitation: invitation2,
	}
	_, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
}

func Test_BasicMsgCmd(t *testing.T) {
	cmd := connection.BasicMsgCmd{
		Cmd: connection.Cmd{
			Cmd:  wallet1Cmd,
			Name: invitation2.ID,
		},
		Message: "test text",
		Sender:  "test sender",
	}
	err := cmd.Validate()
	assert.NoError(t, err)
	_, err = cmd.Exec(os.Stdout)
	assert.NoError(t, err)
}

func Test_SchemaCreateCmd(t *testing.T) {
	ut := time.Now().Unix() - 1558884840
	schemaName := fmt.Sprintf("NEW_SCHEMA_%v", ut)

	sch := &ssi.Schema{
		Name:    schemaName,
		Version: "1.0",
		Attrs:   []string{"email"},
	}
	cmd := schema.CreateCmd{
		Cmd:    wallet1Cmd,
		Schema: sch,
	}
	assert.NoError(t, cmd.Validate())
	r, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
	schR, ok := r.(*schema.CreateResult)
	assert.True(t, ok)
	assert.NotEmpty(t, schR.Schema.ID)
	schemaID = schR.Schema.ID
}

func Test_SchemaGetCmd(t *testing.T) {
	cmd := schema.GetCmd{
		Cmd: wallet1Cmd,
		ID:  schemaID,
	}
	assert.NoError(t, cmd.Validate())
	r2, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
	schR2, ok := r2.(*schema.GetResult)
	assert.True(t, ok)
	assert.NotEmpty(t, schR2.Schema)
}

func Test_CredDefCreate(t *testing.T) {
	cmd := creddef.CreateCmd{
		Cmd:      wallet1Cmd,
		SchemaID: schemaID,
		Tag:      "TAG_99",
	}
	assert.NoError(t, cmd.Validate())
	r, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
	cdR, ok := r.(*creddef.CreateResult)
	assert.True(t, ok)
	assert.NotEmpty(t, cdR.ID)
	credDefID = cdR.ID
}

func Test_CredDefGet(t *testing.T) {
	cmd := creddef.GetCmd{
		Cmd: wallet1Cmd,
		ID:  schemaID,
	}
	assert.NoError(t, cmd.Validate())
	r2, err := cmd.Exec(os.Stdout)
	assert.NoError(t, err)
	schR2, ok := r2.(*creddef.GetResult)
	assert.True(t, ok)
	assert.NotEmpty(t, schR2.CredDef)
}
