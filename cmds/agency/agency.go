package agency

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/findy-network/findy-agent/agent/agency"
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
	_ "github.com/findy-network/findy-wrapper-go/addons/echo" // Install ledger plugins
	_ "github.com/findy-network/findy-wrapper-go/addons/mem"
	"github.com/findy-network/findy-wrapper-go/config"
	"github.com/findy-network/findy-wrapper-go/pool"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type Cmd struct {
	PoolProtocol      uint64
	PoolName          string
	PoolTxnName       string
	WalletName        string
	WalletPwd         string
	StewardSeed       string
	ServiceName       string
	ServiceName2      string
	HostAddr          string
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
}

func (c *Cmd) Validate() error {
	if c.WalletName == "" || c.WalletPwd == "" {
		return errors.New("wallet identification cannot be empty")
	}
	if c.StewardDid == "" && c.StewardSeed == "" {
		return errors.New("steward identification cannot be empty")
	}
	if c.PoolName == "" {
		return errors.New("pool name cannot be empty")
	}
	if c.ServiceName == "" {
		return errors.New("service name  cannot be empty")
	}
	if c.ServiceName2 == "" {
		return errors.New("service name 2 cannot be empty")
	}
	if c.HostAddr == "" {
		return errors.New("host address cannot be empty")
	}
	if c.HostPort == 0 {
		return errors.New("host port cannot be zero")
	}
	if c.HandshakeRegister == "" {
		return errors.New("wallet identification cannot be empty")
	}
	if c.PsmDb == "" {
		return errors.New("psmd database location must be given")
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
	server.BuildHostAddr(c.HostPort)

	return nil
}

func (c *Cmd) Run() (err error) {
	defer err2.Return(&err)

	err2.Check(server.StartHTTPServer(c.ServiceName, c.ServerPort))

	return nil
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
		currentUser, err := user.Current()
		err2.Check(err)
		home := currentUser.HomeDir

		// make sure not use same location for the enclave as for tests!
		sealedBoxPath = filepath.Join(home, ".indy_client/enclave.bolt")
	}

	return enclave.InitSealedBox(sealedBoxPath)
}

func openStewardWallet(did string, serverCmd *Cmd) *cloud.Agent {
	aw := ssi.NewWalletCfg(serverCmd.WalletName, serverCmd.WalletPwd)
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

	if c.HostPort == 0 {
		c.HostPort = c.ServerPort
	}
}

func (c *Cmd) closeAll() {
	enclave.Close()
	// add close psm
	ssi.ClosePool()
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
