package grpc

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	agency2 "github.com/findy-network/findy-agent-api/grpc/agency"
	pb "github.com/findy-network/findy-agent-api/grpc/ops"
	"github.com/findy-network/findy-agent/agent/agency"
	_ "github.com/findy-network/findy-agent/agent/caapi"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/handshake"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	caclient "github.com/findy-network/findy-agent/client"
	"github.com/findy-network/findy-agent/enclave"
	grpcserver "github.com/findy-network/findy-agent/grpc/server"
	_ "github.com/findy-network/findy-agent/protocol/basicmessage"
	_ "github.com/findy-network/findy-agent/protocol/connection"
	_ "github.com/findy-network/findy-agent/protocol/issuecredential"
	_ "github.com/findy-network/findy-agent/protocol/presentproof"
	_ "github.com/findy-network/findy-agent/protocol/trustping"
	"github.com/findy-network/findy-agent/server"
	"github.com/findy-network/findy-grpc/agency/client"
	"github.com/findy-network/findy-grpc/rpc"
	_ "github.com/findy-network/findy-wrapper-go/addons"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/pool"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type TestMode int

const (
	TestModeCI TestMode = iota
	TestModeBuildEnv
	TestModeRunOne
)

type AgentData struct {
	DID        string
	Invitation string
	CredDefID  string
	ConnID     [3]string
}

func (d AgentData) String() string {
	return fmt.Sprintf(`{DID: "%s",
Invitation: "%s",
CredDefID: "%s",
ConnID: [3]string{"%s","%s", "%s"},
},`, d.DID, d.Invitation, d.CredDefID, d.ConnID[0], d.ConnID[1], d.ConnID[2])
}

var (
	testMode = TestModeCI

	lis            = bufconn.Listen(bufSize)
	agents         *[4]AgentData
	emptyAgents    [4]AgentData
	prebuildAgents [4]AgentData
	baseCfg        *rpc.ClientCfg
)

const bufSize = 1024 * 1024

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestMain(m *testing.M) {
	err2.Check(flag.Set("logtostderr", "true"))
	err2.Check(flag.Set("v", "0"))

	prepareBuildOneTest()
	setUp()
	code := m.Run()

	grpcserver.Server.GracefulStop()

	// IF going to start DEBUGGING ONE TEST run first all of the test with no
	// tear down. Then check setUp() and use
	tearDown()

	os.Exit(code)
}

func setUp() {
	defer err2.CatchTrace(func(err error) {
		fmt.Println("error on setup", err)
	})

	calcTestMode()

	if testMode == TestModeRunOne {
		gob := err2.Bytes.Try(ioutil.ReadFile("ONEdata.gob"))
		dto.FromGOB(gob, &prebuildAgents)
		agents = &prebuildAgents
	} else {
		agents = &emptyAgents
	}

	baseCfg = client.BuildClientConnBase("./cert", "what_ever", 0,
		[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

	// obsolete until all of the logs are on glog
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	handshake.RegisterGobs()

	sw := ssi.NewRawWalletCfg("sovrin_steward_wallet", "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")

	exportPath := os.Getenv("TEST_WORKDIR")
	enclaveFile := strLiteral("enclave", ".bolt", -1)
	var sealedBoxPath string
	if len(exportPath) == 0 {
		exportPath = utils.IndyBaseDir()
		sealedBoxPath = filepath.Join(exportPath, ".indy_client/wallet/"+enclaveFile)
	} else {
		sealedBoxPath = enclaveFile
	}
	err2.Check(enclave.InitSealedBox(sealedBoxPath, nil))

	exportPath = filepath.Join(exportPath, "wallets")

	if os.Getenv("CI") == "true" {
		server.ResetEnv(sw, exportPath)
	}

	r := <-pool.SetProtocolVersion(2)
	if r.Err() != nil {
		log.Panicln(r.Err())
	}

	// IF DEBUGGING ONE TEST run first, todo: move cleanup to tear down? make it easier
	if testMode == TestModeRunOne {
		err2.Check(handshake.LoadRegistered(strLiteral("findy", ".json", -1)))
	} else {
		err2.Check(agency.ResetRegistered(strLiteral("findy", ".json", -1)))
	}

	// IF DEBUGGING ONE TEST use always file ledger
	if testMode == TestModeCI {
		ssi.OpenPool("FINDY_MEM_LEDGER")
	} else {
		ssi.OpenPool("FINDY_FILE_LEDGER")
	}

	handshake.SetStewardFromWallet(sw, "Th7MpTaRZVRYnPiabds81Y")

	utils.Settings.SetServiceName(server.TestServiceName)
	utils.Settings.SetServiceName2(server.TestServiceName2)
	utils.Settings.SetHostAddr("http://localhost:8080")
	utils.Settings.SetVersionInfo("testing testing")
	utils.Settings.SetTimeout(1 * time.Hour)
	utils.Settings.SetExportPath(exportPath)

	//utils.Settings.SetCryptVerbose(true)
	utils.Settings.SetLocalTestMode(true)

	err2.Check(psm.Open(strLiteral("Findy", ".bolt", -1))) // this panics if err..

	go grpcserver.Serve(&rpc.ServerCfg{
		PKI:     rpc.LoadPKI("./cert"),
		Port:    0,
		TestLis: lis,
	})

	server.StartTestHTTPServer()
}

func calcTestMode() {
	defer err2.Catch(func(err error) {
		glog.V(20).Infoln(err)
	})

	currentUser, e := user.Current()
	err2.Check(e)
	filename := filepath.Join(currentUser.HomeDir, "/.config/go/env")
	b := err2.Bytes.Try(ioutil.ReadFile(filename))
	if strings.Contains(string(b), "GO111MODULE=off") {
		glog.V(1).Infoln("testMode := TestModeRunOne, go modules off")
		testMode = TestModeRunOne
	}
}

func prepareBuildOneTest() {
	if testMode != TestModeBuildEnv {
		return
	}

	home := utils.IndyBaseDir()
	glog.V(1).Infoln("----- cleaning ----")
	removeFiles(home, "/.indy_client/worker/ONEunit_test_wallet*")
	removeFiles(home, "/.indy_client/worker/ONEemail*")
	removeFiles(home, "/.indy_client/worker/ONEenclave.bolt")
	removeFiles(home, "/.indy_client/wallet/ONEunit_test_wallet*")
	removeFiles(home, "/.indy_client/wallet/ONEemail*")
	if os.Getenv("TEST_WORKDIR") != "" {
		removeFiles(home, "/wallets/*")
	}
	//enclave.WipeSealedBox()
}

func tearDown() {
	if testMode != TestModeCI {
		return
	}

	home := utils.IndyBaseDir()

	removeFiles(home, "/.indy_client/worker/unit_test_wallet*")
	removeFiles(home, "/.indy_client/worker/email*")
	removeFiles(home, "/.indy_client/wallet/unit_test_wallet*")
	removeFiles(home, "/.indy_client/wallet/email*")
	if os.Getenv("TEST_WORKDIR") != "" {
		removeFiles(home, "/wallets/*")
	}
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

func Test_handleAgencyAPI(t *testing.T) {
	for i := 0; i < 2; i++ {
		t.Run(fmt.Sprintf("ping %d", i), func(t *testing.T) {
			conn := client.TryOpen("findy-root", baseCfg)
			ctx := context.Background()
			opsClient := pb.NewDevOpsClient(conn)
			result, err := opsClient.Enter(ctx, &pb.Cmd{
				Type: pb.Cmd_PING,
			})
			assert.NoError(t, err)
			glog.Infoln(i, "result:", result.GetPing())
			assert.NoError(t, conn.Close())
		})
	}
}

func Test_NewOnboarding(t *testing.T) {
	ut := time.Now().Unix() - 1545924840
	walletName := fmt.Sprintf("email%v", ut)
	for i := 0; i < 1; i++ {
		t.Run(fmt.Sprintf("onboard %d", i), func(t *testing.T) {
			conn := client.TryOpen("findy-root", baseCfg)
			ctx := context.Background()
			agencyClient := pb.NewAgencyClient(conn)
			result, err := agencyClient.Onboard(ctx, &pb.Onboarding{
				Email: walletName,
			})
			if assert.NoError(t, err) {
				glog.Infoln(i, "result:", result.GetOk(), result.GetResult().CADID)
			}
			assert.NoError(t, conn.Close())
		})
	}
}

// Test_handshakeAgencyAPI is not actual test here. It's used for the build
// environment for the actual tests. However, it's now used to test that we can
// use only one wallet for all of the EAs. That's handy for web wallets.
func Test_handshakeAgencyAPI_NoOneRun(t *testing.T) {
	if testMode == TestModeRunOne {
		return
	}

	ut := time.Now().Unix() - 1545924840
	schemaName := fmt.Sprintf("NEW_SCHEMA_%v", ut)

	sch := ssi.Schema{
		Name:    schemaName,
		Version: "1.0",
		Attrs:   []string{"email"},
	}

	type args struct {
		wallet ssi.Wallet
		email  string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: strLiteral("unit_test_wallet_grpc", "", -1)},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				email: strLiteral("email", "", 1),
			},
			nil,
		},
		{"2nd",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: strLiteral("unit_test_wallet_grpc", "", -1)},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				email: strLiteral("email", "", 2),
			},
			nil,
		},
		{"third",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: strLiteral("unit_test_wallet_grpc", "", -1)},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				email: strLiteral("email", "", 3),
			},
			nil,
		},
		{"fourth",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: strLiteral("unit_test_wallet_grpc", "", -1)},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				email: strLiteral("email", "", 4),
			},
			nil,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := caclient.Client{
				Email:       tt.args.email,
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			cadid, _, _, err := c.Handshake()
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handshake API = %v, want %v", got, tt.want)
			}
			agents[i].DID = cadid

			// build schema and cred def for the first agent to use later
			if i == 0 {
				sID, err := c.CreateSchema(&sch)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.CreateSchema() %v, want %v", got, tt.want)
				}
				glog.Infoln("==== creating cred def please wait ====")
				time.Sleep(2 * time.Millisecond) // Legacy: Sleep to let ledger process schema!
				agents[0].CredDefID, err = c.CreateCredDef(sID, "TAG_1")
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.CreateCredDef() %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestInvitation_NoOneRun(t *testing.T) {
	if testMode == TestModeRunOne {
		return
	}

	for i, ca := range agents {
		t.Run(fmt.Sprintf("agent%d", i), func(t *testing.T) {
			conn := client.TryOpen(ca.DID, baseCfg)

			ctx := context.Background()
			c := agency2.NewAgentClient(conn)
			r, err := c.CreateInvitation(ctx, &agency2.InvitationBase{Id: utils.UUID()})
			assert.NoError(t, err)

			assert.NotEmpty(t, r.JsonStr)
			glog.Infoln(r.JsonStr)
			agents[i].Invitation = r.JsonStr

			assert.NoError(t, conn.Close())
		})
	}
}

func TestConnection_NoOneRun(t *testing.T) {
	if testMode == TestModeRunOne {
		return
	}

	for i, ca := range agents {
		if i == 0 {
			continue
		}
		t.Run(fmt.Sprintf("agent%d", i), func(t *testing.T) {
			conn := client.TryOpen(agents[0].DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			pairwise := &client.Pairwise{
				Conn:  conn,
				Label: "TestLabel",
			}
			connID, ch, err := pairwise.Connection(ctx, ca.Invitation)
			assert.NoError(t, err)
			assert.NotEmpty(t, connID)
			for status := range ch {
				fmt.Printf("Connection status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			agents[0].ConnID[i-1] = connID
			agents[i].ConnID[0] = connID // must write directly to source not to var 'ca'

			assert.NoError(t, conn.Close())
		})
	}

	for i, agent := range agents {
		glog.Infoln("// agent number:", i)
		glog.Infoln(agent.String())
	}
	if testMode == TestModeBuildEnv {
		err2.Check(ioutil.WriteFile("ONEdata.gob", dto.ToGOB(agents), 0644))
	}
}

func TestTrustPing(t *testing.T) {
	intCh := make(chan struct{})
	if testMode == TestModeRunOne {
		err2.Check(flag.Set("v", "0"))

		go runPSMHook(intCh)
	}

	for i, ca := range agents {
		t.Run(fmt.Sprintf("agent%d", i), func(t *testing.T) {
			conn := client.TryOpen(ca.DID, baseCfg)

			ctx := context.Background()
			commClient := agency2.NewDIDCommClient(conn)
			r, err := client.Pairwise{
				ID:   ca.ConnID[0],
				Conn: conn,
			}.Ping(ctx)
			assert.NoError(t, err)
			var protocolID *agency2.ProtocolID
			for status := range r {
				glog.Infof("trust ping status: %s|%s: %s\n", ca.ConnID[0], status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
				protocolID = status.ProtocolId
			}
			pid, err := commClient.Release(ctx, protocolID)
			assert.NoError(t, err)
			glog.V(1).Infoln("release:", pid.Id)
			assert.NoError(t, conn.Close())
		})
	}
	if testMode == TestModeRunOne {
		intCh <- struct{}{}
	}
}

func runPSMHook(intCh chan struct{}) {
	defer err2.CatchTrace(func(err error) {
		glog.V(1).Infoln("WARNING: error when reading response:", err)
		//close(statusCh)
	})
	conn := client.TryOpen("findy-root", baseCfg)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ch, err := conn.PSMHook(ctx)
	err2.Check(err)
loop:
	for {
		select {
		case status, ok := <-ch:
			if !ok {
				glog.V(1).Infoln("closed from server")
				break loop
			}
			glog.Infoln("\n\t===== listen status:\n\t", status.StatusJson)
		case <-intCh:
			cancel()
			glog.V(1).Infoln("interrupted by user, cancel() called")
		}
	}
}

func TestBasicMessage(t *testing.T) {
	for i, ca := range agents {
		t.Run(fmt.Sprintf("agent_%d", i), func(t *testing.T) {
			conn := client.TryOpen(ca.DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			r, err := client.Pairwise{
				ID:   ca.ConnID[0],
				Conn: conn,
			}.BasicMessage(ctx, "basic message test string")
			assert.NoError(t, err)
			for status := range r {
				glog.V(1).Infof("basic message status: %s|%s: %s\n", ca.ConnID[0], status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})
	}
}

var allPermissive = true

func TestSetPermissive(t *testing.T) {
	for i, ca := range agents {
		conn := client.TryOpen(ca.DID, baseCfg)

		ctx := context.Background()
		c := agency2.NewAgentClient(conn)
		implID := "permissive_sa"
		persistent := false
		if i == 0 && !allPermissive {
			glog.Infoln("--- Using grpc impl ID for SA ---")
			implID = "grpc"
			persistent = true
		}
		r, err := c.SetImplId(ctx, &agency2.SAImplementation{
			Id: implID, Persistent: persistent})
		if t != nil {
			assert.NoError(t, err)
			assert.Equal(t, implID, r.Id)
		}
		err = conn.Close()
		if err != nil && t != nil {
			assert.NoError(t, err)
		}
	}
	glog.Infoln("permissive impl set is done!")
}

// if we don't use auto accept mechanism, we should have listeners for each of
// the receiving agent. Those listeners will accept and offer base to NACK tests
// as well.

func TestIssue(t *testing.T) {
	allPermissive = true
	if testMode == TestModeRunOne {
		TestSetPermissive(t)
	}

	err2.Check(flag.Set("v", "0"))

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("ISSUE-%d", i), func(t *testing.T) {
			conn := client.TryOpen(agents[0].DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			connID := agents[0].ConnID[i]
			r, err := client.Pairwise{
				ID:   connID,
				Conn: conn,
			}.IssueWithAttrs(ctx, agents[0].CredDefID,
				&agency2.Protocol_Attrs{Attrs: []*agency2.Protocol_Attribute{{
					Name:  "email",
					Value: strLiteral("email", "", i+1),
				}}})
			assert.NoError(t, err)
			for status := range r {
				glog.Infof("issuing status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})

	}
}

func TestIssueJSON(t *testing.T) {
	allPermissive = true
	if testMode == TestModeRunOne {
		TestSetPermissive(t)
	}

	err2.Check(flag.Set("v", "0"))

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("ISSUE-%d", i), func(t *testing.T) {
			conn := client.TryOpen(agents[0].DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			connID := agents[0].ConnID[i]
			emailCred := []didcomm.CredentialAttribute{
				{
					Name:  "email",
					Value: strLiteral("email", "", i+1),
				},
			}
			attrJSON := dto.ToJSON(emailCred)
			r, err := client.Pairwise{
				ID:   connID,
				Conn: conn,
			}.Issue(ctx, agents[0].CredDefID, attrJSON)
			assert.NoError(t, err)
			for status := range r {
				glog.Infof("issuing status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})

	}
}

func TestReqProof(t *testing.T) {
	allPermissive = true
	if testMode == TestModeRunOne {
		TestIssue(t)
	}

	err2.Check(flag.Set("v", "0"))

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("PROOF-%d", i), func(t *testing.T) {
			conn := client.TryOpen(agents[0].DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			connID := agents[0].ConnID[i]
			attrs := []*agency2.Protocol_Proof_Attr{{
				Name:      "email",
				CredDefId: agents[0].CredDefID,
			}}
			r, err := client.Pairwise{
				ID:   connID,
				Conn: conn,
			}.ReqProofWithAttrs(ctx, &agency2.Protocol_Proof{Attrs: attrs})
			assert.NoError(t, err)
			for status := range r {
				glog.Infof("proof status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})
	}
}

func TestReqProofJSON(t *testing.T) {
	allPermissive = true
	if testMode == TestModeRunOne {
		TestIssue(t)
	}

	err2.Check(flag.Set("v", "0"))

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("PROOF-%d", i), func(t *testing.T) {
			conn := client.TryOpen(agents[0].DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			connID := agents[0].ConnID[i]
			attrs := []didcomm.ProofAttribute{{
				Name:      "email",
				CredDefID: agents[0].CredDefID,
			}}
			r, err := client.Pairwise{
				ID:   connID,
				Conn: conn,
			}.ReqProof(ctx, dto.ToJSON(attrs))
			assert.NoError(t, err)
			for status := range r {
				glog.Infof("proof status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})
	}
}

func TestListen(t *testing.T) {
	err2.Check(flag.Set("v", "0"))
	waitCh := make(chan struct{})
	intCh := make(chan struct{})
	readyCh := make(chan struct{})
	// start listeners
	for i, ca := range agents {
		if i == 0 {
			continue
		}
		if i == 1 {
			go doListen(ca.DID, intCh, readyCh, waitCh)
		}
	}
	i := 0
	ca := agents[i]
	/*for i, ca := range agents*/ {
		t.Run(fmt.Sprintf("agent_%d", i), func(t *testing.T) {
			conn := client.TryOpen(ca.DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			<-waitCh
			r, err := client.Pairwise{
				ID:   ca.ConnID[0],
				Conn: conn,
			}.BasicMessage(ctx, fmt.Sprintf("# %d. basic message test string", i))
			assert.NoError(t, err)
			for status := range r {
				glog.Infof("basic message status: %s|%s: %s\n", ca.ConnID[0], status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
		})
	}
	<-readyCh           // listener is tested now and it's ready
	intCh <- struct{}{} // tell it to stop

	glog.Infoln("*** closing..")
	time.Sleep(1 * time.Millisecond) // make sure everything is clean after
}

func TestListen100(t *testing.T) {
	err2.Check(flag.Set("v", "0"))
	for i := 0; i < 10; i++ {
		TestListen(t)
	}
}

func BenchmarkIssue(b *testing.B) {
	if testMode == TestModeRunOne {
		TestSetPermissive(nil)
	}

	err2.Check(flag.Set("v", "0"))

	i := 0
	conn := client.TryOpen(agents[0].DID, baseCfg)
	ctx := context.Background()
	agency2.NewDIDCommClient(conn)
	connID := agents[0].ConnID[i]
	// warm up
	{
		r, err := client.Pairwise{
			ID:   connID,
			Conn: conn,
		}.IssueWithAttrs(ctx, agents[0].CredDefID,
			&agency2.Protocol_Attrs{Attrs: []*agency2.Protocol_Attribute{{
				Name:  "email",
				Value: strLiteral("email", "", i+1),
			}}})
		err2.Check(err)
		for range r {
		}
	}
	for n := 0; n < b.N; n++ {
		r, err := client.Pairwise{
			ID:   connID,
			Conn: conn,
		}.IssueWithAttrs(ctx, agents[0].CredDefID,
			&agency2.Protocol_Attrs{Attrs: []*agency2.Protocol_Attribute{{
				Name:  "email",
				Value: strLiteral("email", "", i+1),
			}}})
		err2.Check(err)
		for range r {
		}
	}
	err2.Check(conn.Close())
}

func BenchmarkReqProof(b *testing.B) {
	if testMode == TestModeRunOne {
		TestSetPermissive(nil)
	}

	err2.Check(flag.Set("v", "0"))

	i := 0
	conn := client.TryOpen(agents[0].DID, baseCfg)
	ctx := context.Background()
	agency2.NewDIDCommClient(conn)
	connID := agents[0].ConnID[i]
	// warm up
	{
		attrs := []*agency2.Protocol_Proof_Attr{{
			Name:      "email",
			CredDefId: agents[0].CredDefID,
		}}
		r, err := client.Pairwise{
			ID:   connID,
			Conn: conn,
		}.ReqProofWithAttrs(ctx, &agency2.Protocol_Proof{Attrs: attrs})
		err2.Check(err)
		for range r {
		}
	}
	for n := 0; n < b.N; n++ {
		attrs := []*agency2.Protocol_Proof_Attr{{
			Name:      "email",
			CredDefId: agents[0].CredDefID,
		}}
		r, err := client.Pairwise{
			ID:   connID,
			Conn: conn,
		}.ReqProofWithAttrs(ctx, &agency2.Protocol_Proof{Attrs: attrs})
		err2.Check(err)
		for range r {
		}
	}
	err2.Check(conn.Close())
}

func TestListenSAGrpcProofReq(t *testing.T) {
	allPermissive = false
	TestSetPermissive(t)

	err2.Check(flag.Set("v", "3"))
	waitCh := make(chan struct{})
	intCh := make(chan struct{})
	readyCh := make(chan struct{})
	// start listeners for grpc SA
	for i, ca := range agents {
		if i == 0 {
			go doListen(ca.DID, intCh, readyCh, waitCh)
		}
	}
	i := 0
	ca := agents[i]
	/*for i, ca := range agents*/ {
		t.Run(fmt.Sprintf("agent_%d", i), func(t *testing.T) {
			conn := client.TryOpen(ca.DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			<-waitCh

			connID := ca.ConnID[i]
			attrs := []*agency2.Protocol_Proof_Attr{{
				Name:      "email",
				CredDefId: ca.CredDefID,
			}}
			r, err := client.Pairwise{
				ID:   connID,
				Conn: conn,
			}.ReqProofWithAttrs(ctx, &agency2.Protocol_Proof{Attrs: attrs})
			assert.NoError(t, err)
			for status := range r {
				glog.Infof("proof status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})
	}
	<-readyCh           // listener is tested now and it's ready
	intCh <- struct{}{} // tell it to stop

	glog.Infoln("*** closing..")
	time.Sleep(1 * time.Millisecond) // make sure everything is clean after
}

func TestListenGrpcIssuingResume(t *testing.T) {
	if testMode != TestModeRunOne { // todo: until all tests are ready
		return
	}

	allPermissive = false
	//	TestSetPermissive(t)

	err2.Check(flag.Set("v", "3"))
	waitCh := make(chan struct{})
	intCh := make(chan struct{})
	readyCh := make(chan struct{})
	// start listener for holder
	for i, ca := range agents {
		if i == 1 {
			go doListen(ca.DID, intCh, readyCh, waitCh)
		}
	}
	i := 0
	ca := agents[i]
	/*for i, ca := range agents*/ {
		t.Run(fmt.Sprintf("agent_%d", i), func(t *testing.T) {
			conn := client.TryOpen(ca.DID, baseCfg)

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			<-waitCh

			connID := agents[0].ConnID[i]
			r, err := client.Pairwise{
				ID:   connID,
				Conn: conn,
			}.IssueWithAttrs(ctx, agents[0].CredDefID,
				&agency2.Protocol_Attrs{Attrs: []*agency2.Protocol_Attribute{{
					Name:  "email",
					Value: strLiteral("email", "", i+1),
				}}})
			assert.NoError(t, err)
			for status := range r {
				glog.Infof("issuing status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})
	}
	<-readyCh           // listener is tested now and it's ready
	intCh <- struct{}{} // tell it to stop

	glog.Infoln("*** closing..")
	time.Sleep(1 * time.Millisecond) // make sure everything is clean after
}

func doListen(caDID string, intCh chan struct{}, readyCh chan struct{}, wait chan struct{}) {
	conn := client.TryOpen(caDID, baseCfg)
	//defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := conn.Listen(ctx, &agency2.ClientID{Id: utils.UUID()})
	err2.Check(err)
	glog.V(1).Infoln("** start to listen")
	count := 0
	wait <- struct{}{}
loop:
	for {
		select {
		case status, ok := <-ch:
			if !ok {
				glog.V(1).Infoln("closed from server")
				break loop
			}
			glog.Infoln("\n\t===== listen status:\n\t",
				status.Notification.ConnectionId,
				status.Notification.ProtocolFamily,
				status.Notification.TypeId,
				status.Notification.Id,
				status.Notification.ProtocolId)
			switch status.Notification.TypeId {
			case agency2.Notification_STATUS_UPDATE:
				noAction := handleStatus(conn, status, true)
				if noAction {
					readyCh <- struct{}{}
				}
				count++ // run two times to test our own sending
				if count > 1 {
					readyCh <- struct{}{}
				}
			case agency2.Notification_ACTION_NEEDED:
				resume(conn.ClientConn, status, true)
			case agency2.Notification_ANSWER_NEEDED_PING:
				reply(conn.ClientConn, status, true)
			case agency2.Notification_ANSWER_NEEDED_ISSUE_PROPOSE:
				reply(conn.ClientConn, status, true)
			case agency2.Notification_ANSWER_NEEDED_PROOF_PROPOSE:
				reply(conn.ClientConn, status, true)
			case agency2.Notification_ANSWER_NEEDED_PROOF_VERIFY:
				reply(conn.ClientConn, status, true)
			}
		case <-intCh:
			cancel()
			glog.V(1).Infoln("interrupted by user, cancel() called")
		}
	}
}

func handleStatus(conn client.Conn, status *agency2.AgentStatus, _ bool) bool {
	if status.Notification.ProtocolType == agency2.Protocol_BASIC_MESSAGE {
		ctx := context.Background()
		didComm := agency2.NewDIDCommClient(conn)
		statusResult, err := didComm.Status(ctx, &agency2.ProtocolID{
			TypeId:           status.Notification.ProtocolType,
			Role:             agency2.Protocol_ADDRESSEE,
			Id:               status.Notification.ProtocolId,
			NotificationTime: status.Notification.Timestamp,
		})
		err2.Check(err)
		if statusResult.GetBasicMessage().SentByMe {
			glog.V(1).Infoln("-- ours, no reply")
			return false
		}
		ch, err := client.Pairwise{
			ID:   status.Notification.ConnectionId,
			Conn: conn,
		}.BasicMessage(context.Background(), statusResult.GetBasicMessage().Content)
		err2.Check(err)
		for state := range ch {
			glog.Infoln("BM send state:", state.State, "|", state.Info)
			//assert.Equal(t, agency2.ProtocolState_OK, state.State)
		}
		return false
	}
	return true
}

func reply(conn *grpc.ClientConn, status *agency2.AgentStatus, ack bool) {
	ctx := context.Background()
	c := agency2.NewAgentClient(conn)
	cid, err := c.Give(ctx, &agency2.Answer{
		Id:       status.Notification.Id,
		ClientId: status.ClientId,
		Ack:      ack,
		Info:     "testing says hello!",
	})
	err2.Check(err)
	glog.Infof("Sending the answer (%s) send to client:%s\n", status.Notification.Id, cid.Id)
}

func resume(conn *grpc.ClientConn, status *agency2.AgentStatus, ack bool) {
	ctx := context.Background()
	didComm := agency2.NewDIDCommClient(conn)
	stateAck := agency2.ProtocolState_ACK
	if !ack {
		stateAck = agency2.ProtocolState_NACK
	}
	unpauseResult, err := didComm.Resume(ctx, &agency2.ProtocolState{
		ProtocolId: &agency2.ProtocolID{
			TypeId: status.Notification.ProtocolType,
			Role:   agency2.Protocol_RESUME,
			Id:     status.Notification.ProtocolId,
		},
		State: stateAck,
	})
	err2.Check(err)
	glog.Infoln("======= result:", unpauseResult.String())
}

func strLiteral(prefix string, suffix string, i int) string {
	switch testMode {
	case TestModeCI:
		if i == -1 {
			return prefix + suffix
		}
		return fmt.Sprintf("%s%d%s", prefix, i, suffix)
	case TestModeBuildEnv, TestModeRunOne:
		if i == -1 {
			return "ONE" + prefix + suffix
		}
		// these are used for email literals and they are used for cloud
		// wallet names, these need to be different as well
		return fmt.Sprintf("ONE%s%d%s", prefix, i, suffix)
	default:
		panic("not implemented")
	}
}
