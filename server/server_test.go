package server

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/agency"
	_ "github.com/findy-network/findy-agent/agent/caapi"
	"github.com/findy-network/findy-agent/agent/handshake"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	caclient "github.com/findy-network/findy-agent/client"
	"github.com/findy-network/findy-agent/enclave"
	_ "github.com/findy-network/findy-agent/protocol/basicmessage"
	_ "github.com/findy-network/findy-agent/protocol/connection"
	_ "github.com/findy-network/findy-agent/protocol/issuecredential"
	_ "github.com/findy-network/findy-agent/protocol/presentproof"
	_ "github.com/findy-network/findy-agent/protocol/trustping"
	_ "github.com/findy-network/findy-wrapper-go/addons"
	"github.com/findy-network/findy-wrapper-go/pool"
	"github.com/findy-network/findy-wrapper-go/wallet"
	"github.com/lainio/err2"
)

var endpoint2 service.Addr
var credDefID string

func TestMain(m *testing.M) {
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

	// We don't want logs on file with tests
	err2.Check(flag.Set("logtostderr", "true"))

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
	err2.Check(enclave.InitSealedBox(sealedBoxPath, nil))

	exportPath = filepath.Join(exportPath, "wallets")

	if os.Getenv("CI") == "true" {
		ResetEnv(sw, exportPath)
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

	utils.Settings.SetServiceName(TestServiceName)
	utils.Settings.SetServiceName2(TestServiceName2)
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

	StartTestHTTPServer()
}

func tearDown() {
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
	client := caclient.Client{BaseAddress: "http://localhost:8080"}
	if err := client.ServicePing(); err != nil {
		t.Errorf("ServicePing error: %v", err)
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
					Config: wallet.Config{ID: "unit_test_wallet1"},
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
					Config: wallet.Config{ID: "unit_test_wallet2"},
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
			client := caclient.Client{
				Email:       tt.args.email,
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			_, _, endp, err := client.Handshake()
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handshake API = %v, want %v", got, tt.want)
			}
			if i == 1 {
				endpoint2 = endp
			}
		})
	}
}

func Test_PingCA(t *testing.T) {
	type args struct {
		wallet ssi.Wallet
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			},
			nil,
		},
		{"2nd",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			err := client.PingCA()
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.PingCA() %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_CreatePW(t *testing.T) {
	type args struct {
		wallet ssi.Wallet
		name   string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st PW creation",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name: "pw_name",
			},
			nil,
		},
		{"2nd PW with the same data",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name: "pw_name",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.CreatePW(endpoint2.Endp, endpoint2.Key, tt.args.name)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.CreatePW() %v, want %v", got, tt.want)
			}
			time.Sleep(1 * time.Second)
			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}
		})
	}
}

func TestClient_TrustPingPW(t *testing.T) {
	type args struct {
		wallet ssi.Wallet
		myEndp string
		verkey string
		name   string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name: "pw_name",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.TrustPingPW(tt.args.name)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.TrustPingPW() %v, want %v", got, tt.want)
			}
			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func Test_GetWallet(t *testing.T) {
	type args struct {
		wallet ssi.Wallet
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"2nd",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			wn, err := client.GetWallet()
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("get wallet API = %v, want %v", got, tt.want)
			}
			fmt.Println(wn)
		})
	}
}

func Test_CreateSchemaAndCredDef(t *testing.T) {
	ut := time.Now().Unix() - 1558884840
	schemaName := fmt.Sprintf("NEW_SCHEMA_%v", ut)

	sch := ssi.Schema{
		Name:    schemaName,
		Version: "1.0",
		Attrs:   []string{"email"},
	}

	type args struct {
		wallet ssi.Wallet
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"2nd",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			sID, err := client.CreateSchema(&sch)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.CreateSchema() %v, want %v", got, tt.want)
			}
			time.Sleep(1 * time.Second) // Sleep to let ledger process schema!
			credDefID, err = client.CreateCredDef(sID, "TAG_1")
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.CreateCredDef() %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_SetSAImpl(t *testing.T) {
	type args struct {
		wallet ssi.Wallet
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		//{"1st",
		//	args{
		//		wallet: ssi.Wallet{
		//			Config: wallet.Config{ID: "unit_test_wallet1"},
		//			Credentials: wallet.Credentials{
		//				Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
		//				KeyDerivationMethod: "RAW",
		//			},
		//		},
		//	},
		//	nil,
		//},
		{"2nd",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			err := client.SetSAImpl("email_issuer_verifier")
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.SetSAImpl() %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_CredReq(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
		veriID  string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name",
				emailCr: "name@site.net",
				veriID:  "1000000000000001",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.CredReq(tt.args.name, credDefID, tt.args.emailCr, tt.args.veriID)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.CredReq() %v, want %v", got, tt.want)
			}
			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func TestClient_CredOffer(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name",
				emailCr: "name2@site.net",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientHolder := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet: &ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			}
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}

			tID, err := client.CredOffer(tt.args.name, credDefID, tt.args.emailCr)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.CredOffer() %v, want %v", got, tt.want)
			}

			time.Sleep(3 * time.Second)

			err = clientHolder.TaskStatus(tID)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.TaskStatus() %v, want %v", got, tt.want)
			}

			err = clientHolder.ContinueIssuingProtocol(tID, true)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ContinueProtocol() %v, want %v", got, tt.want)
			}

			err = clientHolder.TaskStatus(tID)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.TaskStatus() %v, want %v", got, tt.want)
			}

			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func TestClient_CredOfferAutoPermissionOn(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name",
				emailCr: "name2@site.net",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientHolder := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet: &ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			}
			err := clientHolder.SetSAImpl("permissive_sa")
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientHolder.SetSAImpl() %v, want %v", got, tt.want)
			}

			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}

			tID, err := client.CredOffer(tt.args.name, credDefID, tt.args.emailCr)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.CredOffer() %v, want %v", got, tt.want)
			}

			time.Sleep(3 * time.Second)

			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					err := clientHolder.SetSAImpl("") // remove auto accept
					if got := err; !reflect.DeepEqual(got, tt.want) {
						t.Errorf("clientHolder.SetSAImpl() %v, want %v", got, tt.want)
					}
					break
				}
			}

		})
	}
}

func TestClient_PwAndTrustPing(t *testing.T) {
	type args struct {
		wallet ssi.Wallet
		myEndp string
		verkey string
		name   string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st totally new name for PW to create",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name: "pw_name_2_tp", // using totally new name for pw
			},
			nil,
		},
		{"2nd use the same PW name but still create a new pairwise",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name: "pw_name_2_tp", // using same name as the test 1
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.PwAndTrustPing(
				endpoint2.Endp, endpoint2.Key, tt.args.name)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.PwAndCredReq() %v, want %v", got, tt.want)
			}
			for {
				time.Sleep(1000 * time.Millisecond)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func TestClient_PwAndCredReq(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
		veriID  string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name_2", // using new new for NEW pw!
				emailCr: "name@site.net",
				veriID:  "1000000000000001",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.PwAndCredReq(
				endpoint2.Endp, endpoint2.Key, tt.args.name, credDefID,
				tt.args.emailCr, tt.args.veriID)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.PwAndCredReq() %v, want %v", got, tt.want)
			}
			time.Sleep(3 * time.Second)
			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func TestClient_ProofProp(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name",
				emailCr: "name@site.net",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.ProofProp(tt.args.name, tt.args.emailCr)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ProofProp() %v, want %v", got, tt.want)
			}
			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func TestClient_ProofRequest(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name",
				emailCr: "name@site.net",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientHolder := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet: &ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			}
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}

			tID, err := client.ProofRequest(tt.args.name, "")
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ProofProp() %v, want %v", got, tt.want)
			}

			time.Sleep(2 * time.Second)
			err = clientHolder.ContinueProtocol(tID, true)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ContinueProtocol() %v, want %v", got, tt.want)
			}

			err = clientHolder.TaskStatus(tID)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.TaskStatus() %v, want %v", got, tt.want)
			}

			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func TestClient_ProofRequestAutoPermissionOn(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet2"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name",
				emailCr: "name@site.net",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientHolder := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet: &ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
			}
			// set auto accept on
			err := clientHolder.SetSAImpl("permissive_sa")
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientHolder.SetSAImpl() %v, want %v", got, tt.want)
			}

			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}

			tID, err := client.ProofRequest(tt.args.name, "")
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.ProofProp() %v, want %v", got, tt.want)
			}

			time.Sleep(2 * time.Second)

			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					err := clientHolder.SetSAImpl("")
					if got := err; !reflect.DeepEqual(got, tt.want) {
						t.Errorf("clientHolder.SetSAImpl() %v, want %v", got, tt.want)
					}

					break
				}
			}

		})
	}
}

func TestClient_PwAndProofProp(t *testing.T) {
	type args struct {
		wallet  ssi.Wallet
		myEndp  string
		verkey  string
		name    string
		emailCr string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				name:    "pw_name_3", // !using new new for NEW pw!
				emailCr: "name@site.net",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.PwAndProofProp(
				endpoint2.Endp, endpoint2.Key, tt.args.name, credDefID, tt.args.emailCr)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.PwAndProofProp() %v, want %v", got, tt.want)
			}
			time.Sleep(2 * time.Second)
			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}

func TestClient_SendMsg(t *testing.T) {
	type args struct {
		wallet   ssi.Wallet
		senderID string
		name     string
		text     string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{"1st",
			args{
				wallet: ssi.Wallet{
					Config: wallet.Config{ID: "unit_test_wallet1"},
					Credentials: wallet.Credentials{
						Key:                 "6cih1cVgRH8yHD54nEYyPKLmdv67o8QbufxaTHot3Qxp",
						KeyDerivationMethod: "RAW",
					},
				},
				senderID: "sender ID 1",
				name:     "pw_name",
				text:     "hello world!",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := caclient.Client{
				BaseAddress: "http://localhost:8080",
				Wallet:      &tt.args.wallet,
			}
			tID, err := client.SendMsg(tt.args.name, tt.args.senderID, tt.args.text)
			if got := err; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("client.SendMsg() %v, want %v", got, tt.want)
			}
			for {
				time.Sleep(1 * time.Second)
				ready, err := client.TaskReady(tID)
				if got := err; !reflect.DeepEqual(got, tt.want) {
					t.Errorf("client.TaskReady() %v, want %v", got, tt.want)
				}
				if ready {
					break
				}
			}

		})
	}
}
