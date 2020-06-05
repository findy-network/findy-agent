package psm

import (
	"os"
	"testing"

	"github.com/go-test/deep"
)

const (
	dbPath = "db_test.bolt"
)

var (
	db *DB
)

func init() {
	os.Remove(dbPath)
	db, _ = OpenDb(dbPath)
}

func Test_addPSM(t *testing.T) {
	psm := testPSM(0)
	tests := []struct {
		name string
	}{
		{"add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.addPSM(psm)
			if err != nil {
				t.Errorf("addPSM() %s error %v", tt.name, err)
			}
			got, err := db.GetPSM(StateKey{DID: mockStateDID, Nonce: mockStateNonce})
			if diff := deep.Equal(psm, got); err != nil || diff != nil {
				t.Errorf("GetPSM() diff %v, err %v", diff, err)
			}
		})
	}
}

func Test_getAllPSM(t *testing.T) {
	registerGobs()
	data := []PSM{*testPSM(0), *testPSM(123), *testPSM(200)}
	for _, d := range data {
		db.addPSM(&d)
	}

	tests := []struct {
		name           string
		sinceTimestamp int64
		data           []PSM
	}{
		{"get all", -1, data},
		{"get since 1", 1, []PSM{data[1], data[2]}},
		{"get since 200", 200, []PSM{data[2]}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tsSince *int64
			if tt.sinceTimestamp >= 0 {
				tsSince = &tt.sinceTimestamp
			}
			got, err := db.AllPSM(mockStateDID, tsSince)
			if diff := deep.Equal(tt.data, *got); err != nil || diff != nil {
				t.Errorf("getAllPSM() diff %v, err %v", diff, err)
			}
		})
	}
}

func Test_getAllDeviceID(t *testing.T) {
	deviceIDRep := &DeviceIDRep{
		DID:         mockStateDID,
		DeviceToken: "token",
	}
	data := []DeviceIDRep{*deviceIDRep}
	for _, d := range data {
		db.AddDeviceIDRep(&d)
	}

	tests := []struct {
		name string
		data []DeviceIDRep
	}{
		{"get all", data},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GetAllDeviceIDRep(mockStateDID)
			if diff := deep.Equal(tt.data, *got); err != nil || diff != nil {
				t.Errorf("GetAllDeviceIDRep() diff %v, err %v", diff, err)
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
			err := db.AddPairwiseRep(pwRep)
			if err != nil {
				t.Errorf("AddPairwiseRep() %s error %v", tt.name, err)
			}
			got, err := db.GetPairwiseRep(pwRep.Key)
			if diff := deep.Equal(pwRep, got); err != nil || diff != nil {
				t.Errorf("GetPairwiseRep() diff %v, err %v", diff, err)
			}
		})
	}
}

func Test_addBasicMessageRep(t *testing.T) {
	msgRep := &BasicMessageRep{
		Key:    StateKey{DID: mockStateDID, Nonce: mockStateNonce},
		PwName: "pwName",
	}
	tests := []struct {
		name string
	}{
		{"add"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.AddBasicMessageRep(msgRep)
			if err != nil {
				t.Errorf("AddBasicMessageRep() %s error %v", tt.name, err)
			}
			got, err := db.GetBasicMessageRep(msgRep.Key)
			if diff := deep.Equal(msgRep, got); err != nil || diff != nil {
				t.Errorf("GetBasicMessageRep() diff %v, err %v", diff, err)
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
			err := db.AddIssueCredRep(credRep)
			if err != nil {
				t.Errorf("AddIssueCredRep() %s error %v", tt.name, err)
			}
			got, err := db.GetIssueCredRep(credRep.Key)
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
			err := db.AddPresentProofRep(proofRep)
			if err != nil {
				t.Errorf("AddPresentProofRep() %s error %v", tt.name, err)
			}
			got, err := db.GetPresentProofRep(proofRep.Key)
			if diff := deep.Equal(proofRep, got); err != nil || diff != nil {
				t.Errorf("GetPresentProofRep() diff %v, err %v", diff, err)
			}
		})
	}
}
