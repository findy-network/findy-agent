package storage

import (
	"flag"
	"os"
	"sync"
	"testing"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/lainio/err2/try"
	"github.com/stretchr/testify/require"
)

type storageTest struct {
	name    string
	config  api.AgentStorageConfig
	storage api.AgentStorage
}

var (
	kmsTestStorages []*storageTest
)

const (
	nameAfgo = "afgo"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	try.To(flag.Set("logtostderr", "true"))
	try.To(flag.Set("stderrthreshold", "WARNING"))
	try.To(flag.Set("v", "10"))
	flag.Parse()

	kmsTestStorages = make([]*storageTest, 0)
	// AFGO
	kmsTestConfig := api.AgentStorageConfig{
		AgentKey: mgddb.GenerateKey(),
		AgentID:  "agentID",
		FilePath: ".",
	}
	afgoTestStorage, err := mgddb.New(kmsTestConfig)
	if err != nil {
		panic(err)
	}
	kmsTestStorages = append(kmsTestStorages, &storageTest{
		name:    nameAfgo,
		config:  kmsTestConfig,
		storage: afgoTestStorage,
	})
}

func tearDown() {
	for _, testStorage := range kmsTestStorages {
		if err := testStorage.storage.Close(); err != nil {
			panic(err)
		}
		// afgo
		os.RemoveAll(testStorage.config.AgentID + ".bolt")
	}
}

func TestConcurrentOpen(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.storage.Close()
			require.NoError(t, err)

			wg := &sync.WaitGroup{}
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					err := testCase.storage.Open()
					require.NoError(t, err)

					store := testCase.storage.KMS()
					keyID, keyBytes, err := store.CreateAndExportPubKeyBytes(kms.ED25519Type)
					require.NoError(t, err)
					require.NotEmpty(t, keyID)
					require.NotEmpty(t, keyBytes)
				}()
			}
			wg.Wait()
		})
	}
}

func TestDIDStore(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.DIDStorage()
			testDID := api.DID{
				ID:  "did:test:123",
				DID: "did:test:123",
			}
			err := store.AddDID(testDID)
			require.NoError(t, err)

			gotDID, err := store.GetDID(testDID.ID)
			require.NoError(t, err)
			require.Equal(t, testDID, *gotDID)
		})
	}
}

func TestConnectionStore(t *testing.T) {
	for index := range kmsTestStorages {
		testCase := kmsTestStorages[index]
		t.Run(testCase.name, func(t *testing.T) {
			store := testCase.storage.ConnectionStorage()
			testConn := api.Connection{
				ID:            "123-uid",
				OurDID:        "did:test:123",
				TheirDID:      "did:test:456",
				TheirEndpoint: "https://example.com",
				TheirRoute:    []string{"routeKey"},
			}
			err := store.AddConnection(testConn)
			require.NoError(t, err)

			gotConn, err := store.GetConnection(testConn.ID)
			require.NoError(t, err)
			require.Equal(t, testConn, *gotConn)

			testConn2 := testConn
			testConn2.ID = "456-uid"
			err = store.AddConnection(testConn2)
			require.NoError(t, err)

			conns, err := store.ListConnections()
			require.NoError(t, err)
			require.Len(t, conns, 2)
			require.Equal(t, testConn, conns[0])
			require.Equal(t, testConn2, conns[1])
		})
	}
}
