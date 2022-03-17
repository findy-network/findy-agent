package vdr

import (
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"os"
	"testing"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdr"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name     string
	registry vdr.VDR
}

var (
	storageTestConfig = api.AgentStorageConfig{
		AgentKey: mgddb.GenerateKey(),
		AgentID:  "agentID",
		FilePath: ".",
	}
	afgoTestStorage *mgddb.Storage
	tests           []testCase
	testKey         []byte
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

	// AFGO
	var err error
	afgoTestStorage, err = mgddb.New(storageTestConfig)
	assert.D.True(err == nil)
	assert.D.True(afgoTestStorage != nil)

	testVdr, err := New(afgoTestStorage)
	assert.D.True(err == nil)
	assert.D.True(testVdr != nil)

	tests = append(
		tests,
		testCase{
			name:     "key",
			registry: testVdr.Key(),
		},
		testCase{
			name:     "peer",
			registry: testVdr.Peer(),
		},
	)

	testKey, _, err = ed25519.GenerateKey(rand.Reader)
	assert.D.True(err == nil)
}

func tearDown() {
	err := afgoTestStorage.Close()
	assert.D.True(err == nil)

	os.RemoveAll(storageTestConfig.AgentID + ".bolt")
}

func TestVDRAccept(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, tt.registry.Accept(tt.name))
			require.False(t, tt.registry.Accept("invalid"))
		})
	}
}

func TestVDRCreateAndRead(t *testing.T) {
	doc := &did.Doc{
		VerificationMethod: []did.VerificationMethod{{
			Type:  "Ed25519VerificationKey2018",
			Value: testKey,
		}},
		Service: []did.Service{{
			Type:            "DidcCommServiceType",
			ServiceEndpoint: "http://example.com",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docResolution, err := tt.registry.Create(doc)
			require.NoError(t, err)
			require.NotNil(t, docResolution)

			docResolution, err = tt.registry.Read(docResolution.DIDDocument.ID)
			require.NoError(t, err)
			require.NotNil(t, docResolution)
		})
	}
}

func TestVDRUpdate(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.registry.Update(nil).Error(), "not supported")
		})
	}
}

func TestVDRDeactivate(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.registry.Deactivate("").Error(), "not supported")
		})
	}
}

func TestVDRClose(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Nil(t, tt.registry.Close())
		})
	}
}
