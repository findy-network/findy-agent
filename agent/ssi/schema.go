package ssi

import (
	"encoding/json"

	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/findy-network/findy-wrapper-go/ledger"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type Schema struct {
	ID      string   `json:"id,omitempty"`      // ID from Indy/Ledger
	Name    string   `json:"name,omitempty"`    // name of the schema
	Version string   `json:"version,omitempty"` // version number in string
	Attrs   []string `json:"attrs,omitempty"`   // attribute string list
	Stored  *Future  `json:"-"`                 // info from ledger
}

func (s *Schema) Create(DID string) (err error) {
	defer err2.Annotate("create schema", &err)
	attrsStr, err := json.Marshal(s.Attrs)
	try.To(err)

	s.Stored = &Future{}
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
	return ledger.WriteSchema(Pool(), wallet, DID, scJSON)
}

func CredDefFromLedger(DID, credDefID string) (cd string, err error) {
	defer err2.Annotate("process get cred def", &err)

	_, cd, err = ledger.ReadCredDef(Pool(), DID, credDefID)
	return cd, err
}

func (s *Schema) FromLedger(DID string) (err error) {
	defer err2.Annotate("schema from ledger", &err)

	sID, schema, err := ledger.ReadSchema(Pool(), DID, s.ValidID())
	try.To(err)
	s.Stored = &Future{V: dto.Result{Data: dto.Data{Str1: sID, Str2: schema}}, On: Consumed}

	return nil
}

func (s *Schema) LazySchema() string {
	if s.Stored == nil {
		return ""
	}
	return s.Stored.Str2()
}
