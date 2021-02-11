package agency

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/accessmgr"
	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/apns"
	_ "github.com/findy-network/findy-agent/agent/caapi" // Command handlers need these
	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/handshake"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/enclave"
	_ "github.com/findy-network/findy-agent/protocol/basicmessage" // protocols needed
	_ "github.com/findy-network/findy-agent/protocol/connection"
	_ "github.com/findy-network/findy-agent/protocol/issuecredential"
	_ "github.com/findy-network/findy-agent/protocol/notification"
	_ "github.com/findy-network/findy-agent/protocol/presentproof"
	_ "github.com/findy-network/findy-agent/protocol/trustping"
	"github.com/findy-network/findy-agent/server"
	_ "github.com/findy-network/findy-wrapper-go/addons" // Install ledger plugins
	"github.com/findy-network/findy-wrapper-go/config"
	"github.com/findy-network/findy-wrapper-go/pool"
	"github.com/go-co-op/gocron"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

type Cmd struct {
	PoolProtocol      uint64
	PoolName          string
	WalletName        string
	WalletPwd         string
	StewardSeed       string
	ServiceName       string
	ServiceName2      string
	HostAddr          string
	HostScheme        string
	HostPort          uint
	ServerPort        uint
	ExportPath        string
	EnclavePath       string
	StewardDid        string
	HandshakeRegister string
	PsmDb             string
	ResetData         bool
	URL               string
	VersionInfo       string
	Salt              string
	APNSP12CertFile   string
	AllowRPC          bool
	GRPCPort          int
	TlsCertPath       string
	JWTSecret         string

	EnclaveKey        string
	EnclaveBackupName string
	EnclaveBackupTime string

	RegisterBackupName     string
	RegisterBackupInterval time.Duration

	WalletBackupPath string
	WalletBackupTime string

	GRPCAdmin string
}

var (
	cron = gocron.NewScheduler(time.Now().Location())

	DefaultValues = Cmd{
		PoolProtocol:           2,
		PoolName:               "findy-pool",
		WalletName:             "",
		WalletPwd:              "",
		StewardSeed:            "000000000000000000000000Steward1",
		ServiceName:            "ca-api",
		ServiceName2:           "a2a",
		HostAddr:               "localhost",
		HostScheme:             "http",
		HostPort:               8080,
		ServerPort:             8080,
		ExportPath:             "",
		EnclavePath:            "",
		StewardDid:             "",
		HandshakeRegister:      "findy.json",
		PsmDb:                  "findy.bolt",
		ResetData:              false,
		URL:                    "",
		VersionInfo:            "",
		Salt:                   "",
		APNSP12CertFile:        "",
		AllowRPC:               true,
		GRPCPort:               50051,
		TlsCertPath:            "",
		JWTSecret:              "",
		EnclaveKey:             "",
		EnclaveBackupName:      "",
		EnclaveBackupTime:      "",
		RegisterBackupName:     "",
		RegisterBackupInterval: 0,
		WalletBackupPath:       "",
		WalletBackupTime:       "",
		GRPCAdmin:              "findy-root",
	}
)

func (c *Cmd) Validate() (err error) {
	defer err2.Return(&err)

	c.SetMustHaveDefaults()

	assert.P.NotEmpty(c.HostScheme, "host scheme cannot be empty")
	assert.P.True(c.WalletName != "" && c.WalletPwd != "", "wallet identification cannot be empty")
	assert.P.True(!(c.StewardDid == "" && c.StewardSeed == ""), "steward identification cannot be empty")
	assert.P.NotEmpty(c.PoolName, "pool name cannot be empty")
	assert.P.NotEmpty(c.ServiceName, "service name  cannot be empty")
	assert.P.NotEmpty(c.ServiceName2, "service name 2 cannot be empty")
	assert.P.NotEmpty(c.HostAddr, "host address cannot be empty")
	assert.P.True(c.HostPort != 0, "host port cannot be zero")
	assert.P.NotEmpty(c.PsmDb, "psmd database location must be given")
	assert.P.NotEmpty(c.HandshakeRegister, "handshake register path cannot be empty")
	if c.RegisterBackupName == "" {
		glog.Warning("handshake register backup should be empty in production")
	}
	if c.EnclaveBackupName == "" {
		glog.Warning("enclave backup shouldn't be empty in production")
	}
	if c.WalletBackupPath == "" {
		glog.Warning("wallet backup path shouldn't be empty in production")
	}
	if c.WalletBackupTime != "" {
		if err := cmds.ValidateTime(c.WalletBackupTime); err != nil {
			return err
		}
	}
	if c.EnclaveBackupTime != "" {
		if err := cmds.ValidateTime(c.EnclaveBackupTime); err != nil {
			return err
		}
	}
	if c.APNSP12CertFile != "" {
		_, err := os.Stat(c.APNSP12CertFile)
		if os.IsNotExist(err) {
			return errors.New("apns p12 cert file does not exist")
		}
	}
	return nil
}

func (c *Cmd) Exec(_ io.Writer) (r cmds.Result, err error) {
	return nil, StartAgency(c)
}

func (c *Cmd) Setup() (err error) {
	defer err2.Return(&err)

	c.printStartupArgs()
	err2.Check(c.initSealedBox())
	c.startLoadingAgents()
	err2.Check(psm.Open(c.PsmDb))
	ssi.OpenPool(c.PoolName)
	c.checkSteward()
	c.setRuntimeSettings()
	server.BuildHostAddr(c.HostScheme, c.HostPort)

	if c.APNSP12CertFile != "" {
		utils.Settings.SetCertFileForAPNS(c.APNSP12CertFile)
		err2.Check(apns.Init())
	}
	return nil
}

func (c *Cmd) Run() (err error) {
	defer err2.Return(&err)

	c.startBackupTasks()
	if c.AllowRPC {
		StartGrpcServer(c.GRPCPort, c.TlsCertPath, c.JWTSecret)
	}
	err2.Check(server.StartHTTPServer(c.ServiceName, c.ServerPort))

	return nil
}

func (c *Cmd) startBackupTasks() {
	if c.WalletBackupPath != "" {
		accessmgr.Start() // start the wallet backup tracker

		glog.V(1).Infoln("wallet backup time:", c.WalletBackupTime)
		_, err := cron.Every(1).Day().At(c.WalletBackupTime).Do(accessmgr.StartBackup)
		if err != nil {
			glog.Warningln("wallet backup start error:", err)
		}
	}
	if c.EnclaveBackupName != "" {
		glog.V(1).Infoln("enclave backup time:", c.EnclaveBackupTime)
		_, err := cron.Every(1).Day().At(c.EnclaveBackupTime).Do(enclave.Backup)
		if err != nil {
			glog.Warningln("enclave backup start error:", err)
		}
	}
	if c.RegisterBackupName != "" {
		_, err := cron.Every(1).Day().At("04:30").Do(agency.Backup)
		if err != nil {
			glog.Warningln("register backup start error:", err)
		}
	}

	cron.StartAsync()
}

func StartAgency(serverCmd *Cmd) (err error) {
	defer err2.Return(&err)

	err2.Check(serverCmd.Setup())
	err2.Check(serverCmd.Run())
	serverCmd.closeAll()

	return nil
}

func (c *Cmd) initSealedBox() (err error) {
	defer err2.Return(&err)

	sealedBoxPath := c.EnclavePath
	if sealedBoxPath == "" {
		home := utils.IndyBaseDir()

		// make sure not use same location for the enclave as for tests!
		sealedBoxPath = filepath.Join(home, ".indy_client/enclave.bolt")
	}

	return enclave.InitSealedBox(
		sealedBoxPath, c.EnclaveBackupName, c.EnclaveKey)
}

func openStewardWallet(did string, serverCmd *Cmd) *cloud.Agent {
	aw := ssi.NewRawWalletCfg(serverCmd.WalletName, serverCmd.WalletPwd)
	a := cloud.Agent{}
	a.OpenWallet(*aw)
	a.SetRootDid(a.OpenDID(did))
	return &a
}

func (c *Cmd) PreRun() {
	utils.Settings.SetVersionInfo(c.VersionInfo)
	config.Set(config.SystemConfig{CryptoThreadPoolSize: 8})
	setProtocol(c.PoolProtocol)

	handshake.RegisterGobs()

	if c.Salt == "" {
		saltFromEnv := os.Getenv("FINDY_AGENT_SALT")
		if len(saltFromEnv) > 0 {
			utils.Salt = saltFromEnv
		}
	} else {
		utils.Salt = c.Salt
	}
}

func setProtocol(version uint64) {
	r := <-pool.SetProtocolVersion(version)
	if r.Err() != nil {
		fmt.Println(r.Err())
		panic(r.Err())
	}
}

func (c *Cmd) printStartupArgs() {
	fmt.Println(
		"HandshakeRegister path:", c.HandshakeRegister,
		"\nState machine db path:", c.PsmDb,
		"\nHost address:", c.HostAddr,
		"\nHost port:", c.HostPort,
		"\nServer port:", c.ServerPort)
}

func (c *Cmd) startLoadingAgents() {
	if c.ResetData {
		err2.Check(agency.ResetRegistered(c.HandshakeRegister))
	} else {
		err2.Check(handshake.LoadRegistered(c.HandshakeRegister))
	}
}

func (c *Cmd) checkSteward() {
	var steward *cloud.Agent
	if c.StewardSeed != "" && c.StewardDid == "" {
		glog.Fatal("cannot start without steward")
	} else if c.WalletName != "" && c.WalletPwd != "" {
		steward = openStewardWallet(c.StewardDid, c)
	}
	handshake.SetSteward(steward)
}

func (c *Cmd) setRuntimeSettings() {
	utils.Settings.SetServiceName(c.ServiceName)
	utils.Settings.SetServiceName2(c.ServiceName2)
	utils.Settings.SetHostAddr(c.HostAddr)
	utils.Settings.SetExportPath(c.ExportPath)
	utils.Settings.SetWalletBackupPath(c.WalletBackupPath)
	utils.Settings.SetWalletBackupTime(c.WalletBackupTime)
	utils.Settings.SetRegisterBackupName(c.RegisterBackupName)
	utils.Settings.SetRegisterBackupInterval(c.RegisterBackupInterval)
	utils.Settings.SetGRPCAdmin(c.GRPCAdmin)

	if c.HostPort == 0 {
		c.HostPort = c.ServerPort
	}
}

func (c *Cmd) closeAll() {
	enclave.Close()
	// add close psm
	ssi.ClosePool()
}

func (c *Cmd) SetMustHaveDefaults() {
	if c.HostScheme == "" {
		glog.V(5).Infoln("setting default scheme to HTTP")
		c.HostScheme = DefaultValues.HostScheme
	}
	if c.GRPCAdmin == "" {
		glog.V(5).Infoln("setting default to admin id")
		c.GRPCAdmin = DefaultValues.GRPCAdmin
	}
}

func ParseLoggingArgs(s string) {
	args := make([]string, 1, 12)
	args[0] = os.Args[0]
	args = append(args, strings.Split(s, " ")...)
	orgArgs := os.Args
	os.Args = args
	flag.Parse()
	os.Args = orgArgs
}
