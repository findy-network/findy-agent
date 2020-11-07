package grpc

import (
	"context"
	"flag"
	"fmt"
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
	"github.com/findy-network/findy-wrapper-go/pool"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/lainio/err2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var (
	credDefID string
	CADID     [2]string
	lis     = bufconn.Listen(bufSize)
)

const bufSize = 1024 * 1024

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestMain(m *testing.M) {
	err2.Check(flag.Set("logtostderr", "true"))
	err2.Check(flag.Set("v", "0"))
	setUp()
	code := m.Run()
	// IF going to start DEBUGGING ONE TEST run first all of the test with no
	// tear down. Then check setUp() and use
	tearDown()
	os.Exit(code)
}

func setUp() {
	defer err2.CatchTrace(func(err error) {
		fmt.Println("error on setup", err)
	})

	// obsolete until all of the logs are on glog
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	handshake.RegisterGobs()

	sw := ssi.NewRawWalletCfg("sovrin_steward_wallet", "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")

	exportPath := os.Getenv("TEST_WORKDIR")
	var sealedBoxPath string
	if len(exportPath) == 0 {
		exportPath = utils.IndyBaseDir()
		sealedBoxPath = filepath.Join(exportPath, ".indy_client/wallet/enclave.bolt")
	} else {
		sealedBoxPath = "enclave.bolt"
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

	// IF DEBUGGING ONE TEST run first
	err2.Check(agency.ResetRegistered("findy.json"))
	//hubReady, err := Hub().LoadRegistered("findy.json")

	ssi.OpenPool("FINDY_MEM_LEDGER")

	handshake.SetStewardFromWallet(sw, "Th7MpTaRZVRYnPiabds81Y")

	utils.Settings.SetServiceName(server.TestServiceName)
	utils.Settings.SetServiceName2(server.TestServiceName2)
	utils.Settings.SetHostAddr("http://localhost:8080")
	utils.Settings.SetVersionInfo("testing testing")
	utils.Settings.SetTimeout(1 * time.Hour)
	utils.Settings.SetExportPath(exportPath)

	//utils.Settings.SetCryptVerbose(true)
	utils.Settings.SetLocalTestMode(true)

	err2.Check(psm.Open("Findy.bolt")) // this panics if err..

	// IF DEBUGGING ONE TEST run first without next line, then when using
	// LoadRegistered() in the beginning the line is needed.
	//<-hubReady

	go grpcserver.Serve(lis)

	server.StartTestHTTPServer()
}

func tearDown() {
	grpcserver.Server.GracefulStop()

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

func Test_handshakeAgencyAPI(t *testing.T) {
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
					Config: wallet.Config{ID: "unit_test_wallet_grpc1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				email: "email1",
			},
			nil,
		},
		{"2nd",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet_grpc2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				email: "email2",
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
			CADID[i] = cadid
			//if i == 1 {
			//	endpoint2 = endp
			//}
		})
	}
}

func TestInvitation(t *testing.T) {
	for _, ca := range CADID {
		conn := client.TryOpenConn(ca, "", 50051,
			[]grpc.DialOption{grpc.WithContextDialer(bufDialer)})

		ctx := context.Background()
		c := agency2.NewAgentClient(conn)
		r, err := c.CreateInvitation(ctx, &agency2.InvitationBase{Id: utils.UUID()})
		assert.NoError(t, err)

		assert.NotEmpty(t, r.JsonStr)
		fmt.Println(r.JsonStr)

		assert.NoError(t, conn.Close())
		fmt.Println(ca)
	}
}
