package psm

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/findy-network/findy-common-go/crypto/db"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/go-test/deep"
	"github.com/lainio/err2"
	"github.com/stretchr/testify/assert"
)

const (
	dbPath = "db_test.bolt"
)

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	tearDown()
	os.Exit(code)
}

func setUp() {
	defer err2.CatchTrace(func(err error) {
		fmt.Println("error on setup", err)
	})

	// We don't want logs on file with tests
	err2.Check(flag.Set("logtostderr", "true"))

	err2.Check(Open(dbPath))
}

func tearDown() {
	db.Close()

	os.Remove(dbPath)
}

func Test_addPSM(t *testing.T) {
	psm := testPSM(0)
	assert.NotNil(t, psm)

	tests := []struct {
		name string
	}{
		{"add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddPSM(psm)
			if err != nil {
				t.Errorf("AddPSM() %s error %v", tt.name, err)
			}
			got, err := GetPSM(StateKey{DID: mockStateDID, Nonce: mockStateNonce})
			if diff := deep.Equal(psm, got); err != nil || diff != nil {
				t.Errorf("GetPSM() diff %v, err %v", diff, err)
			}
		})
	}
}

func Test_addPairwiseRep(t *testing.T) {
	pwRep := &PairwiseRep{
		Key:  StateKey{DID: mockStateDID, Nonce: mockStateNonce},
		Name: "name",
	}
	tests := []struct {
		name string
	}{
		{"add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddPairwiseRep(pwRep)
			if err != nil {
				t.Errorf("AddPairwiseRep() %s error %v", tt.name, err)
			}
			got, err := GetPairwiseRep(pwRep.Key)
			if diff := deep.Equal(pwRep, got); err != nil || diff != nil {
				t.Errorf("GetPairwiseRep() diff %v, err %v", diff, err)
			}
		})
	}
}

type TestRep struct {
	BaseRep
}

func (t *TestRep) Type() byte {
	return BucketPSM
}

func NewTestRep(d []byte) Rep {
	p := &TestRep{}
	dto.FromGOB(d, p)
	return p
}

func Test_addBaseRep(t *testing.T) {
	msgRep := &TestRep{
		BaseRep: BaseRep{Key: StateKey{DID: mockStateDID, Nonce: mockStateNonce, Type: BucketPSM}},
	}
	tests := []struct {
		name string
	}{
		{"add"},
	}
	Creator.Add(BucketPSM, NewTestRep)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddRep(msgRep)
			if err != nil {
				t.Errorf("AddRep() %s error %v", tt.name, err)
			}
			got, err := GetRep(msgRep.Key)
			if diff := deep.Equal(msgRep, got); err != nil || diff != nil {
				t.Errorf("GetRep() diff %v, err %v", diff, err)
			}
		})
	}
}

func Test_addIssueCredRep(t *testing.T) {
	credRep := &IssueCredRep{
		Key:       StateKey{DID: mockStateDID, Nonce: mockStateNonce},
		Timestamp: 123,
	}
	tests := []struct {
		name string
	}{
		{"add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddIssueCredRep(credRep)
			if err != nil {
				t.Errorf("AddIssueCredRep() %s error %v", tt.name, err)
			}
			got, err := GetIssueCredRep(credRep.Key)
			if diff := deep.Equal(credRep, got); err != nil || diff != nil {
				t.Errorf("GetIssueCredRep() diff %v, err %v", diff, err)
			}
		})
	}
}

func Test_addPresentProofRep(t *testing.T) {
	proofRep := &PresentProofRep{
		Key:      StateKey{DID: mockStateDID, Nonce: mockStateNonce},
		ProofReq: "proofReq",
	}
	tests := []struct {
		name string
	}{
		{"add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddPresentProofRep(proofRep)
			if err != nil {
				t.Errorf("AddPresentProofRep() %s error %v", tt.name, err)
			}
			got, err := GetPresentProofRep(proofRep.Key)
			if diff := deep.Equal(proofRep, got); err != nil || diff != nil {
				t.Errorf("GetPresentProofRep() diff %v, err %v", diff, err)
			}
		})
	}
}
