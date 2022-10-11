package cfg

import (
	"flag"
	"os"
	"testing"

	"github.com/findy-network/findy-agent/agent/storage/api"
	"github.com/findy-network/findy-agent/agent/storage/mgddb"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

var (
	_ = AgentStorage{AgentStorageConfig: api.AgentStorageConfig{
		AgentKey: "",
		AgentID:  "",
		FilePath: "",
	}}

	testConfig = []AgentStorage{
		{
			api.AgentStorageConfig{
				AgentKey: mgddb.GenerateKey(),
				AgentID:  "agentID_1",
				FilePath: ".",
			},
		},
		{
			api.AgentStorageConfig{
				AgentKey: mgddb.GenerateKey(),
				AgentID:  "agentID_2",
				FilePath: ".",
			},
		},
	}
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
}

func tearDown() {
	for _, cfg := range testConfig {
		_ = os.RemoveAll(cfg.AgentID + ".bolt")
	}
}

func TestAgentStorageConfig_OpenWallet(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()
	for round := 0; round < 2; round++ {
		handles := make([]int, len(testConfig))
		for i, cfg := range testConfig {
			h, err := cfg.OpenWallet()
			assert.NoError(err)
			assert.NotEqual(-1, h)

			handles[i] = h
			oldH := h
			h, err = cfg.OpenWallet()
			assert.NoError(err)
			assert.Equal(oldH, h)

			oldH = h
			h, err = cfg.OpenWallet()
			assert.NoError(err)
			assert.Equal(oldH, h)

			aStorage := Storage(h)
			assert.INotNil(aStorage)
		}
		for i, cfg := range testConfig {
			err := cfg.CloseWallet(handles[i])
			assert.NoError(err)
		}
	}
}
