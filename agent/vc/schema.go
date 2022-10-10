package vc

import (
	"encoding/json"

	"github.com/findy-network/findy-agent/agent/async"
	"github.com/findy-network/findy-agent/agent/pool"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	indyDto "github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/ledger"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type Schema struct {
	ID      string        `json:"id,omitempty"`      // ID from Indy/Ledger
	Name    string        `json:"name,omitempty"`    // name of the schema
	Version string        `json:"version,omitempty"` // version number in string
	Attrs   []string      `json:"attrs,omitempty"`   // attribute string list
	Stored  *async.Future `json:"-"`                 // info from ledger
}

func (s *Schema) Create(DID string) (err error) {
	defer err2.Returnf(&err, "create schema by DID (%v)", DID)

	attrsStr := try.To1(json.Marshal(s.Attrs))

	s.Stored = &async.Future{}
	s.Stored.SetChan(anoncreds.IssuerCreateSchema(DID, s.Name, s.Version, string(attrsStr)))
	return err
}

func (s *Schema) ValidID() string {
	if s.ID != "" {
		return s.ID
	}
	if s.Stored != nil {
		s.ID = s.Stored.Str1()
	}
	return s.ID
}

func (s *Schema) ToLedger(wallet int, DID string) error {
	scJSON := s.Stored.Str2()
	return ledger.WriteSchema(pool.Handle(), wallet, DID, scJSON)
}

func CredDefFromLedger(DID, credDefID string) (cd string, err error) {
	defer err2.Returnf(&err, "process get cred def")

	_, cd, err = ledger.ReadCredDef(pool.Handle(), DID, credDefID)
	return cd, err
}

func (s *Schema) FromLedger(DID string) (err error) {
	defer err2.Returnf(&err, "schema from ledger")

	sID, schema := try.To2(ledger.ReadSchema(pool.Handle(), DID, s.ValidID()))
	s.Stored = &async.Future{V: indyDto.Result{Data: indyDto.Data{Str1: sID, Str2: schema}}, On: async.Consumed}

	return nil
}

func (s *Schema) LazySchema() string {
	if s.Stored == nil {
		return ""
	}
	return s.Stored.Str2()
}
