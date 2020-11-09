package grpc

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	agency2 "github.com/findy-network/findy-agent-api/grpc/agency"
	pb "github.com/findy-network/findy-agent-api/grpc/ops"
	"github.com/findy-network/findy-agent/agent/agency"
	_ "github.com/findy-network/findy-agent/agent/caapi"
	"github.com/findy-network/findy-agent/agent/handshake"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	caclient "github.com/findy-network/findy-agent/client"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/findy-network/findy-agent/grpc/client"
	grpcserver "github.com/findy-network/findy-agent/grpc/server"
	_ "github.com/findy-network/findy-agent/protocol/basicmessage"
	_ "github.com/findy-network/findy-agent/protocol/connection"
	_ "github.com/findy-network/findy-agent/protocol/issuecredential"
	_ "github.com/findy-network/findy-agent/protocol/presentproof"
	_ "github.com/findy-network/findy-agent/protocol/trustping"
	"github.com/findy-network/findy-agent/server"
	_ "github.com/findy-network/findy-wrapper-go/addons"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/pool"
	"github.com/findy-network/findy-wrapper-go/wallet"
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
	testMode = TestModeRunOne

	lis            = bufconn.Listen(bufSize)
	agents         *[4]AgentData
	emptyAgents    [4]AgentData
	prebuildAgents = [4]AgentData{
		// agent number: 0
		{DID: "32MxmRhhBzLKY297DNu82m",
			Invitation: `{"serviceEndpoint":"http://localhost:8080/a2a/32MxmRhhBzLKY297DNu82m/32MxmRhhBzLKY297DNu82m/SnhcjPawVGtdGHc7mkJib3","recipientKeys":["26y2UMUMDipr6NhLjMBc3gRXgPfnqZKt4gh8SrBDMCvG"],"@id":"ed04ace9-903a-43b8-9c77-7107ee55b12b","label":"empty-label","@type":"did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/invitation"}`,
			CredDefID:  "2Kc7X1ErDwNQC3mDSzcj2r:3:CL:2Kc7X1ErDwNQC3mDSzcj2r:2:NEW_SCHEMA_58906353:1.0:TAG_1",
			ConnID:     [3]string{"60a5c061-f686-4ed1-9fac-6fda102c0585", "30d6b72b-b812-4781-b564-89a5d885d14c", "9563ab5a-27f8-4842-bbc8-0844d85f5881"},
		},
		// agent number: 1
		{DID: "SDNULvUU932rrYzgXjBgSn",
			Invitation: `{"serviceEndpoint":"http://localhost:8080/a2a/SDNULvUU932rrYzgXjBgSn/SDNULvUU932rrYzgXjBgSn/4QbXMoSb6pJbaUwCRfCTwr","recipientKeys":["Ek3QUrUGXCaqujsFK3HgziAR8JcGJs4bwPYacWtpBFMa"],"@id":"60a5c061-f686-4ed1-9fac-6fda102c0585","label":"empty-label","@type":"did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/invitation"}`,
			CredDefID:  "",
			ConnID:     [3]string{"60a5c061-f686-4ed1-9fac-6fda102c0585", "", ""},
		},
		// agent number: 2
		{DID: "SjAHWNsaE1HwzdtzocRrhN",
			Invitation: `{"serviceEndpoint":"http://localhost:8080/a2a/SjAHWNsaE1HwzdtzocRrhN/SjAHWNsaE1HwzdtzocRrhN/F5aaLV7fyjxt4ph1XXPifo","recipientKeys":["F2H84VhkTb42RUM2YLhTZDTKQdHqN2B3zFJGMDCQKjgJ"],"@id":"30d6b72b-b812-4781-b564-89a5d885d14c","label":"empty-label","@type":"did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/invitation"}`,
			CredDefID:  "",
			ConnID:     [3]string{"30d6b72b-b812-4781-b564-89a5d885d14c", "", ""},
		},
		// agent number: 3
		{DID: "Ki6tgbpTsM1tgP7Qoo21dJ",
			Invitation: `{"serviceEndpoint":"http://localhost:8080/a2a/Ki6tgbpTsM1tgP7Qoo21dJ/Ki6tgbpTsM1tgP7Qoo21dJ/5vEE67yWoRkY7VCpo71wjm","recipientKeys":["BCRCb7J2RrduLnbPfGauG8iUaTg95zvKPLJYNdcXhv3V"],"@id":"9563ab5a-27f8-4842-bbc8-0844d85f5881","label":"empty-label","@type":"did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/connections/1.0/invitation"}`,
			CredDefID:  "",
			ConnID:     [3]string{"9563ab5a-27f8-4842-bbc8-0844d85f5881", "", ""},
		},
	}
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

	if testMode == TestModeRunOne {
		gob := err2.Bytes.Try(ioutil.ReadFile("ONEdata.gob"))
		dto.FromGOB(gob, &prebuildAgents)
		agents = &prebuildAgents
	} else {
		agents = &emptyAgents
	}

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
	err2.Check(enclave.InitSealedBox(sealedBoxPath))

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

	go grpcserver.Serve(lis)

	server.StartTestHTTPServer()
}

func prepareBuildOneTest() {
	if testMode != TestModeBuildEnv {
		return
	}

	home := utils.IndyBaseDir()
	fmt.Println("----- cleaning ----")
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
			conn, err := client.OpenClientConn("findy-root", "what_ever",
				[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})
			assert.NoError(t, err)

			ctx := context.Background()

			opsClient := pb.NewDevOpsClient(conn)
			result, err := opsClient.Enter(ctx, &pb.Cmd{
				Type: pb.Cmd_PING,
			})
			assert.NoError(t, err)
			fmt.Println(i, "result:", result.GetPing())

			assert.NoError(t, conn.Close())
		})
	}
}

// Test_handshakeAgencyAPI is not actual test here. It's used for the build
// environment for the actual tests. However, it's now used to test that we can
// use only one wallet for all of the EAs. That's handy for web wallets.
func Test_handshakeAgencyAPI(t *testing.T) {
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
				fmt.Println("==== creating cred def please wait ====")
				time.Sleep(2 * time.Millisecond) // Legacy: Sleep to let ledger process schema!
				agents[0].CredDefID, err = c.CreateCredDef(sID, "TAG_1")
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.CreateCredDef() %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestInvitation(t *testing.T) {
	if testMode == TestModeRunOne {
		return
	}

	for i, ca := range agents {
		t.Run(fmt.Sprintf("agent%d", i), func(t *testing.T) {
			conn := client.TryOpenConn(ca.DID, "", 50051,
				[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

			ctx := context.Background()
			c := agency2.NewAgentClient(conn)
			r, err := c.CreateInvitation(ctx, &agency2.InvitationBase{Id: utils.UUID()})
			assert.NoError(t, err)

			assert.NotEmpty(t, r.JsonStr)
			fmt.Println(r.JsonStr)
			agents[i].Invitation = r.JsonStr

			assert.NoError(t, conn.Close())
		})
	}
}

func TestConnection(t *testing.T) {
	if testMode == TestModeRunOne {
		return
	}

	for i, ca := range agents {
		if i == 0 {
			continue
		}
		t.Run(fmt.Sprintf("agent%d", i), func(t *testing.T) {
			conn := client.TryOpenConn(agents[0].DID, "", 50051,
				[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			connID, ch, err := client.Connection(ctx, ca.Invitation)
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
		fmt.Println("// agent number:", i)
		fmt.Println(agent.String())
	}
	if testMode == TestModeBuildEnv {
		err2.Check(ioutil.WriteFile("ONEdata.gob", dto.ToGOB(agents), 0644))
	}
}

func TestTrustPing(t *testing.T) {
	for i, ca := range agents {
		t.Run(fmt.Sprintf("agent%d", i), func(t *testing.T) {
			conn := client.TryOpenConn(ca.DID, "", 50051,
				[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			r, err := client.Pairwise{
				ID: ca.ConnID[0],
			}.Ping(ctx)
			assert.NoError(t, err)
			for status := range r {
				fmt.Printf("trust ping status: %s|%s: %s\n", ca.ConnID[0], status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})
	}
}

func TestBasicMessage(t *testing.T) {
	for i, ca := range agents {
		t.Run(fmt.Sprintf("agent_%d", i), func(t *testing.T) {
			conn := client.TryOpenConn(ca.DID, "", 50051,
				[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			r, err := client.Pairwise{
				ID: ca.ConnID[0],
			}.BasicMessage(ctx, "basic message test string")
			assert.NoError(t, err)
			for status := range r {
				fmt.Printf("basic message status: %s|%s: %s\n", ca.ConnID[0], status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})
	}
}

func TestSetPermissive(t *testing.T) {
	for _, ca := range agents {
		conn := client.TryOpenConn(ca.DID, "", 50051,
			[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

		ctx := context.Background()
		c := agency2.NewAgentClient(conn)
		r, err := c.SetImplId(ctx, &agency2.SAImplementation{Id: "permissive_sa"})
		assert.NoError(t, err)
		assert.Equal(t, "permissive_sa", r.Id)
		assert.NoError(t, conn.Close())
	}
	fmt.Println("permissive impl set is done!")
}

// if we don't use auto accept mechanism, we should have listeners for each of
// the receiving agent. Those listeners will accept and offer base to NACK tests
// as well.

func TestIssue(t *testing.T) {
	if testMode == TestModeRunOne {
		TestSetPermissive(t)
	}

	err2.Check(flag.Set("v", "0"))

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("ISSUE-%d", i), func(t *testing.T) {
			conn := client.TryOpenConn(agents[0].DID, "", 50051,
				[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			connID := agents[0].ConnID[i]
			r, err := client.Pairwise{
				ID: connID,
			}.IssueWithAttrs(ctx, agents[0].CredDefID,
				&agency2.Protocol_Attrs{Attrs: []*agency2.Protocol_Attribute{{
					Name:  "email",
					Value: strLiteral("email", "", i+1),
				}}})
			assert.NoError(t, err)
			for status := range r {
				fmt.Printf("issuing status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})

	}
}

func TestReqProof(t *testing.T) {
	if testMode == TestModeRunOne {
		TestIssue(t)
	}

	err2.Check(flag.Set("v", "0"))

	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("PROOF-%d", i), func(t *testing.T) {
			conn := client.TryOpenConn(agents[0].DID, "", 50051,
				[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

			ctx := context.Background()
			agency2.NewDIDCommClient(conn)
			connID := agents[0].ConnID[i]
			attrs := []*agency2.Protocol_Proof_Attr{{
				Name:      "email",
				CredDefId: agents[0].CredDefID,
			}}
			r, err := client.Pairwise{
				ID: connID,
			}.ReqProofWithAttrs(ctx, &agency2.Protocol_Proof{Attrs: attrs})
			assert.NoError(t, err)
			for status := range r {
				fmt.Printf("proof status: %s|%s: %s\n", connID, status.ProtocolId, status.State)
				assert.Equal(t, agency2.ProtocolState_OK, status.State)
			}
			assert.NoError(t, conn.Close())
		})

	}
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

// todo: get one test mode as persistent by starting it to take safe to wallets and other files by renaming them!!
// current directory:
// -rw------- 1 harri harri 2.3M Nov  9 15:45 Findy.bolt
// -rw-r--r-- 1 harri harri  290 Nov  8 12:26 findy.json
// wallet directory:
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 email1//
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 email2//
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 email3//
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 email4//
//        2 -rw------- 1 harri harri  32K Nov  9 15:45 enclave.bolt
// single EA wallet:
//       3 drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 unit_test_wallet_grpc/

// worker-dir:
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 ../worker/email4_worker/
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 ../worker/email3_worker/
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 ../worker/email2_worker/
// drwxrwxr-x 2 harri harri 4.0K Nov  8 12:26 ../worker/email1_worker/
// todo: should we have tests for protocol Start and Status not only Run
//  try to first write first round of tests and then write rest of them

// new API
// create schema, USE old
// create cred def for first agent, USE old

// issue cred for rest of the agents
//	- test listening, approval, ...
// req proof from rest of the agents
//	- test listening, approval, ...

// chat bot stuff, state machine
