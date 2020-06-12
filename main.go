package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	agencyCmd "github.com/findy-network/findy-agent/cmds/agency"
	"github.com/findy-network/findy-agent/cmds/agent"
	"github.com/findy-network/findy-agent/cmds/agent/creddef"
	schema2 "github.com/findy-network/findy-agent/cmds/agent/schema"
	"github.com/findy-network/findy-agent/cmds/connection"
	keycmd "github.com/findy-network/findy-agent/cmds/key"
	"github.com/findy-network/findy-agent/cmds/onboard"
	"github.com/findy-network/findy-agent/cmds/pool"
	stewardCmd "github.com/findy-network/findy-agent/cmds/steward"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

var versionInfo = "OP Tech Lab - Findy Agency v. " + utils.Version

type runMode int
type verb int

const (
	unknown runMode = iota // reserved for not used -cases
	server                 // server mode which continues running until stopped
	client                 // contacts to server to perform certain procedures
	create                 // for creating local resources like wallets, DIDs, pools, etc.
	version                // shows the version number
	help                   // show the full version of the usage
)

func (rm runMode) String() string {
	return [...]string{"unknown", "server", "client", "create", "version", "help"}[rm]
}

var (
	currentRunMode = unknown
)

var (
	currentVerb = nope
)

/*
Subcommands of the service. Subcommands are entered to service like docker's
subcommands are entered to docker. When service is started from the the prompt,
the subcommand is entered as a third argument of the program. Each subcommand
has it's own set of flags which include rest of the parameters. Subcommands
execute the service's commands in certain mode.
*/
const (
	nope verb = iota // reserved for not used -cases
	handshake
	handshakeAndExport
	pairwise
	listen
	send
	ping
	schema
	credDef
	cnx
	steward
	theirDid
	did
	root
	key
	invite
)

func (v verb) String() string {
	return [...]string{"nope", "handshake", "handshakeAndExport", "pairwise", "listen",
		"send", "ping", "schema", "credDef", "cnx", "steward", "theirDid",
		"did", "root", "key", "invite"}[v]
}

/*
Commands of the service. Commands are entered to service like git's commands are
entered to git. When service is started from the the prompt, the command is
entered as a second argument of the program. Commands start the service in
certain running mode.
*/
var (
	startServerCmd = flag.NewFlagSet("server", flag.ExitOnError)
	clientCmd      = flag.NewFlagSet("client", flag.ExitOnError)
	createCmd      = flag.NewFlagSet("create", flag.ExitOnError)
)

// common flags
var serverCmd = &agencyCmd.Cmd{VersionInfo: versionInfo}
var loggingFlags string

// At the moment we use package init() only for initializing flag variables.
// Please be noted that many of the flag variables can be shared by multiple sub
// commands. We try to follow principle: 2 or more share the same flag we share
// the variable as well.
func init() {
	startServerCmd.Uint64Var(&serverCmd.PoolProtocol, "proto", 2, "ledger protocol version")
	startServerCmd.StringVar(&serverCmd.PoolName, "pool", "FINDY_MEM_LEDGER", "name of the pool config")
	startServerCmd.StringVar(&serverCmd.WalletName, "wallet", "", "steward wallet name, password needed")
	startServerCmd.StringVar(&serverCmd.WalletPwd, "pwd", "", "steward wallet password")
	startServerCmd.StringVar(&serverCmd.StewardSeed, "seed", "", "seed for steward DID to be created to new steward wallet")
	startServerCmd.StringVar(&serverCmd.ServiceName, "name", agency.CAAPIPath, "URL path for CA API.")
	startServerCmd.StringVar(&serverCmd.ServiceName2, "a2a", agency.ProtocolPath, "URL path for A2A protocols")
	startServerCmd.StringVar(&loggingFlags, "logging", "-logtostderr=true -v=2", "logging startup arguments")
	startServerCmd.StringVar(&serverCmd.HostAddr, "hostaddr", "localhost", "server name to seen from internet like host.network.com, see -hostport")
	startServerCmd.UintVar(&serverCmd.HostPort, "hostport", 8080, "Host port")
	startServerCmd.UintVar(&serverCmd.ServerPort, "port", 8080, "HTTP server's port")
	startServerCmd.StringVar(&serverCmd.ExportPath, "updir", "static", "upload path for direct file loadings")

	startServerCmd.StringVar(&serverCmd.StewardDid, "did", "", "Steward DID, aka Root DID for Steward Rights to create new Trust Agents")
	startServerCmd.StringVar(&serverCmd.HandshakeRegister, "register", "findy.json", "handshake registry's filename")
	startServerCmd.StringVar(&serverCmd.PsmDb, "psmdb", "findy.bolt", "state machine db's filename")
	startServerCmd.BoolVar(&serverCmd.ResetData, "reset", false, "WARNING! Resets registries, used for the start overs, mainly for testing purposes")

	// for client mode as well
	clientCmd.StringVar(&serverCmd.WalletName, "wallet", "", "wallet name, password needed")
	clientCmd.StringVar(&serverCmd.WalletPwd, "pwd", "", "wallet password")
	clientCmd.StringVar(&serverCmd.URL, "url", "", "url of the Findy Agency")
	clientCmd.StringVar(&serverCmd.ExportPath, "exportpath", "./exported", "local wallet export path")
	clientCmd.StringVar(&loggingFlags, "logging", "-logtostderr=true -v=2", "logging startup arguments")

	// for the create command
	createCmd.StringVar(&loggingFlags, "logging", "-logtostderr=true -v=2", "logging startup arguments")
	createCmd.Uint64Var(&serverCmd.PoolProtocol, "proto", 2, "ledger protocol version")
	createCmd.StringVar(&serverCmd.PoolName, "pool", "FINDY_MEM_LEDGER", "name of the pool config")
	createCmd.StringVar(&serverCmd.WalletName, "wallet", "", "steward wallet name, password needed")
	createCmd.StringVar(&serverCmd.WalletPwd, "pwd", "", "steward wallet KEY (generate)")
	createCmd.StringVar(&serverCmd.StewardSeed, "seed", "", "seed for steward DID to be created to new steward wallet")
	createCmd.StringVar(&serverCmd.URL, "url", "", "url of the Findy Agency")
}

// Client specific flags.
var (
	email = clientCmd.String("email", "", "email of the client registering to Findy Agency")
	pwURL = clientCmd.String("endp", "", "endpoint URL of the pairwise DID we are binding")
	ID    = clientCmd.String("id", "", "ID use for building requests like getCredDef")
	text  = clientCmd.String("text", "", "text message to send connection")
	pw    = clientCmd.String("pw", "", "pairwise connection id")
)

// Create specific flags
var (
	schemaName    = createCmd.String("sc_name", "", "Schema name")
	schemaVersion = createCmd.String("sc_version", "", "Schema version")
	schemaAttrs   = createCmd.String("sc_attrs", "[\"email\"]", "Schema attributes as JSON array")
	schemaID      = createCmd.String("sc_id", "", "Schema ID got from schema creation")
	credDefTag    = createCmd.String("cred_def_tag", "", "Schema credential definition tag")
	poolTxnName   = createCmd.String("txn", "", "name of pool txn file from pool config will be created")
)

func normalMain() {
	defer err2.Catch(func(err error) {
		glog.Error(err)
		glog.Flush()
	})

	serverCmd.PreRun()

	switch currentRunMode {
	case server:
		err2.Check(serverCmd.Validate())
		err2.Check(agencyCmd.StartAgency(serverCmd))
	case client:
		runClient()
	case create:
		runCreate()
	case version:
		fmt.Println(versionInfo)
	case help:
		usage()
	default:
		glog.Warning("unknown run mode")
	}
}

func runCreate() {
	defer err2.Catch(func(err error) {
		fmt.Println(err)
	})

	if serverCmd.URL != "" {
		switch currentVerb {
		case schema:
			var attrs []string
			err2.Check(json.Unmarshal([]byte(*schemaAttrs), &attrs))
			cmd := schema2.CreateCmd{
				Cmd: WalletCmd(),
				Schema: &ssi.Schema{
					Name:    *schemaName,
					Version: *schemaVersion,
					Attrs:   attrs,
				},
			}
			err2.Try(cmd.ValidID())
			err2.Try(cmd.Exec(os.Stdout))
		case credDef:
			cmd := creddef.CreateCmd{
				Cmd:      WalletCmd(),
				SchemaID: *schemaID,
				Tag:      *credDefTag,
			}
			err2.Try(cmd.Validate())
			err2.Try(cmd.Exec(os.Stdout))
		}
	} else {
		switch currentVerb {
		case key:
			cmd := keycmd.CreateCmd{
				Seed: serverCmd.StewardSeed,
			}
			err2.Try(cmd.Validate())
			err2.Try(cmd.Exec(os.Stdout))
		case cnx:
			poolCreateCmd := pool.CreateCmd{
				Name: serverCmd.PoolName,
				Txn:  *poolTxnName,
			}
			err2.Check(poolCreateCmd.Validate())
			err2.Try(poolCreateCmd.Exec(os.Stdout))

		case steward:
			createStewardCmd := stewardCmd.CreateCmd{
				Cmd: cmds.Cmd{
					WalletName: serverCmd.WalletName,
					WalletKey:  serverCmd.WalletPwd,
				},
				PoolName:    serverCmd.PoolName,
				StewardSeed: serverCmd.StewardSeed,
			}
			err2.Check(createStewardCmd.Validate())
			err2.Try(createStewardCmd.Exec(os.Stdout))

		case ping:
			poolPingCmd := pool.PingCmd{
				Name: serverCmd.PoolName,
			}
			err2.Check(poolPingCmd.Validate())
			err2.Try(poolPingCmd.Exec(os.Stdout))
		default:
			cmds.Fprintln(os.Stderr, "not enough information to execute command")
			os.Exit(1)
		}
	}
}

func runClient() {
	defer err2.Catch(func(err error) {
		fmt.Println(err)
		os.Exit(1)
	})

	if serverCmd.WalletName != "" && serverCmd.WalletPwd != "" {
		switch currentVerb {
		case credDef:
			cmd := creddef.GetCmd{
				Cmd: WalletCmd(),
				ID:  *ID,
			}
			err2.Try(cmd.Validate())
			err2.Try(cmd.Exec(os.Stdout))
		case schema:
			cmd := schema2.GetCmd{
				Cmd: WalletCmd(),
				ID:  *ID,
			}
			err2.Try(cmd.Validate())
			err2.Try(cmd.Exec(os.Stdout))
		case send:
			cmd := connection.BasicMsgCmd{
				Cmd: connection.Cmd{
					Cmd:  WalletCmd(),
					Name: *pw,
				},
				Message: *text,
				Sender:  "CLI",
			}
			err2.Try(cmd.Validate())
			err2.Try(cmd.Exec(os.Stdout))
		case ping:
			cmd := agent.PingCmd{
				Cmd: WalletCmd(),
			}
			err2.Try(cmd.Validate())
			err2.Try(cmd.Exec(os.Stdout))
		case invite:
			cmd := agent.InvitationCmd{
				Cmd: WalletCmd(),
			}
			err2.Try(cmd.Validate())
			err2.Try(cmd.Exec(os.Stdout))
		case pairwise:
			var i didexchange.Invitation
			dto.FromJSONStr(*pwURL, &i)
			i.ID = utils.UUID()
			connectionCmd := agent.ConnectionCmd{
				Cmd:        WalletCmd(),
				Invitation: i,
			}
			err2.Try(connectionCmd.Validate())
			err2.Try(connectionCmd.Exec(os.Stdout))

		case handshake:
			onboardCmd := onboard.Cmd{
				Cmd:        WalletCmd(),
				Email:      *email,
				AgencyAddr: serverCmd.URL,
			}
			err2.Try(onboardCmd.Validate())
			err2.Try(onboardCmd.Exec(os.Stdout))
		case handshakeAndExport:
			onboardCmd := onboard.Cmd{
				Cmd:        WalletCmd(),
				Email:      *email,
				AgencyAddr: serverCmd.URL,
			}
			err2.Try(onboardCmd.Validate())
			err2.Try(onboardCmd.Exec(os.Stdout))
			exportCmd := agent.ExportCmd{
				Cmd:       onboardCmd.Cmd,
				Filename:  serverCmd.ExportPath,
				ExportKey: serverCmd.WalletPwd,
			}
			err2.Try(exportCmd.Validate())
			err2.Try(exportCmd.Exec(os.Stdout))
		}
	} else {
		switch {
		case currentVerb == ping:
			agencyPingCmd := agencyCmd.PingCmd{BaseAddr: serverCmd.URL}
			err2.Try(agencyPingCmd.Validate())
			err2.Try(agencyPingCmd.Exec(os.Stdout))
		default:
			fmt.Println("Cannot start client, not enough information")
		}
	}
}

func WalletCmd() cmds.Cmd {
	return cmds.Cmd{
		WalletName: serverCmd.WalletName,
		WalletKey:  serverCmd.WalletPwd,
	}
}

func processArgs() {
	defer err2.Catch(func(err error) {
		fmt.Println(err)
		usage()
	})
	if len(os.Args) < 2 {
		shortUsage()
		os.Exit(0)
	} else if os.Args[1] == "-version" {
		fmt.Println(versionInfo)
		os.Exit(0)
	}
	switch os.Args[1] {
	case startServerCmd.Name():
		_ = startServerCmd.Parse(os.Args[2:])
		currentRunMode = server
	case clientCmd.Name():
		if !subCmdVerb() {
			_, _ = fmt.Fprintf(startServerCmd.Output(),
				"subcommands: handshake, pairwise, listen, ping, invite, send\n")
			os.Exit(1)
		}
		_ = clientCmd.Parse(os.Args[3:])
		currentRunMode = client
	case createCmd.Name():
		if !subCmdVerb() {
			_, _ = fmt.Fprintf(startServerCmd.Output(),
				"subcommands: cnx, schema, did, key, steward, credDef\n")
			os.Exit(1)
		}
		_ = createCmd.Parse(os.Args[3:])
		currentRunMode = create
	case "version":
		currentRunMode = version
	case "help":
		currentRunMode = help
	default:
		shortUsage()
	}
}

func subCmdVerb() (ok bool) {
	if len(os.Args) < 3 {
		return false
	}
	switch os.Args[2] {
	case root.String():
		currentVerb = root
	case key.String():
		currentVerb = key
	case schema.String():
		currentVerb = schema
	case credDef.String():
		currentVerb = credDef
	case cnx.String():
		currentVerb = cnx
	case steward.String():
		currentVerb = steward
	case theirDid.String():
		currentVerb = theirDid
	case did.String():
		currentVerb = did
	case ping.String():
		currentVerb = ping
	case handshake.String():
		currentVerb = handshake
	case handshakeAndExport.String():
		currentVerb = handshakeAndExport
	case pairwise.String():
		currentVerb = pairwise
	case listen.String():
		currentVerb = listen
	case send.String():
		currentVerb = send
	case invite.String():
		currentVerb = invite
	}
	ok = currentVerb != nope
	return ok
}

func usageLn(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(startServerCmd.Output(), format, a...)
}

func shortUsage() string {
	usageLn(versionInfo)
	appName := filepath.Base(os.Args[0])
	usageLn("\nUsage:\t%s %s|%s|%s|help|version <cmd> <flags>\n\n",
		appName, startServerCmd.Name(), clientCmd.Name(), createCmd.Name())
	return appName
}

func usage() {
	appName := shortUsage()

	usageLn("Example:%s %s -pool \"pool1\" -wallet \"wallet1\" -key \"xx..xx\" -did \"steward...DID\"\n",
		appName, startServerCmd.Name())
	usageLn("\t\twhich starts the server with existing Steward wallet and DID\n\n")

	usageLn("Example:%s %s cnx -pool \"pool1\" -txn \"pool_txn_file\"\n",
		appName, createCmd.Name())
	usageLn("\t\twhich creates a new pool config from file pool_txn_file\n\n")

	cmds.Fprintf(startServerCmd.Output(), "usage of %s:\n", startServerCmd.Name())
	startServerCmd.PrintDefaults()
	cmds.Fprintf(startServerCmd.Output(), "\nusage of %s <cmd>:\n", clientCmd.Name())
	cmds.Fprintf(startServerCmd.Output(), "cmd: handshake, pairwise, listen, ping, send\n")
	clientCmd.PrintDefaults()
	cmds.Fprintf(startServerCmd.Output(), "\nusage of %s:\n", createCmd.Name())
	cmds.Fprintf(startServerCmd.Output(), "cmd: cnx, schema, did, steward, credDef\n")
	createCmd.PrintDefaults()
}

func main() {
	//includes flag.Parse() and run mode calculations
	processArgs()

	agencyCmd.ParseLoggingArgs(loggingFlags)

	normalMain()
}
