package signature_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/method"
	"github.com/findy-network/findy-agent/std/didexchange/signature"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func tearDown() {
	home := utils.IndyBaseDir()
	removeFiles(home, "/.indy_client/wallet/signature-test-agent*")
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

var (
	agent, agent2 = new(ssi.DIDAgent), new(ssi.DIDAgent)
)

func setUp() {
	err2.SetTracers(os.Stderr)
	assert.D = assert.AsserterCallerInfo
	assert.DefaultAsserter = assert.AsserterFormattedCallerInfo

	// first, create agent 1 with the storages
	walletID := fmt.Sprintf("signature-test-agent-11%d", time.Now().Unix())
	aw := ssi.NewRawWalletCfg(walletID, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw.Create()
	agent.OpenWallet(*aw)

	// second, create agent 2 with the storages
	walletID2 := fmt.Sprintf("signature-test-agent-12%d", time.Now().Unix())
	aw2 := ssi.NewRawWalletCfg(walletID2, "4Vwsj6Qcczmhk2Ak7H5GGvFE1cQCdRtWfW4jchahNUoE")
	aw2.Create()
	agent2.OpenWallet(*aw2)
}

func TestSignVerifyWithSeparatedWallets(t *testing.T) {

	tests := []struct {
		name         string
		method       method.Type
		createOutDID func(core.DID) (core.DID, error)
	}{
		{"key", method.TypeKey, func(inDID core.DID) (core.DID, error) { return agent2.NewOutDID(inDID.URI()) }},
		{"sov", method.TypeSov, func(inDID core.DID) (core.DID, error) { return agent2.NewOutDID(inDID.URI(), inDID.VerKey()) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PushTester(t)
			defer assert.PopTester()
			defer err2.Catch(func(err error) {
				utils.Settings.SetDIDMethod(method.TypeSov)
			})
			utils.Settings.SetDIDMethod(tt.method)

			didIn, _ := agent.NewDID(tt.method, "")
			assert.INotNil(didIn)
			println("in: ", didIn.URI())

			// give agent1's in-DID (signer) to agent2's out-DID (verifier)
			didOut2, err := tt.createOutDID(didIn)
			assert.NoError(err)
			assert.INotNil(didOut2)
			println("out2: ", didOut2.URI())

			signer := signature.Signer{DID: didIn}
			message := []byte("message")
			signatureData, err := signer.Sign(message)
			assert.NoError(err)

			verifier := signature.Verifier{DID: didOut2}
			assert.NoError(verifier.Verify(message, signatureData))
		})
	}
}
