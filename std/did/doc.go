package did

import (
	"time"

	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/core"
)

// Doc DID Document definition
type Doc struct {
	Context        string               `json:"@context,omitempty"`
	ID             string               `json:"id,omitempty"`
	PublicKey      []PublicKey          `json:"publicKey,omitempty"`
	Service        []Service            `json:"service,omitempty"`
	Authentication []VerificationMethod `json:"authentication,omitempty"`
	Created        *time.Time           `json:"created,omitempty"`
	Updated        *time.Time           `json:"updated,omitempty"`
	//	Proof          []Proof
}

// PublicKey DID doc public key
type PublicKey struct {
	ID              string `json:"id,omitempty"`
	Type            string `json:"type,omitempty"`
	Controller      string `json:"controller,omitempty"`
	PublicKeyBase58 string `json:"publicKeyBase58,omitempty"`
	//Value      []byte `json:"value,omitempty"`
}

// Service DID doc service
type Service struct {
	ID              string                 `json:"id,omitempty"`
	Type            string                 `json:"type,omitempty"`
	Priority        uint                   `json:"priority,omitempty"`
	RecipientKeys   []string               `json:"recipientKeys,omitempty"`
	RoutingKeys     []string               `json:"routingKeys,omitempty"`
	ServiceEndpoint string                 `json:"serviceEndpoint"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}

// VerificationMethod authentication verification method
type VerificationMethod struct {
	Type      string `json:"type,omitempty"`
	PublicKey string `json:"publicKey,omitempty"`
	//	PublicKey `json:"public_key,omitempty"`
}

func NewDoc(did core.DID, ae service.Addr) *Doc {
	didURI := did.String()
	didURIRef := didURI + "#1"
	pubK := PublicKey{
		ID:              didURIRef,
		Type:            "Ed25519VerificationKey2018",
		Controller:      didURI,
		PublicKeyBase58: did.VerKey(),
	}
	service := Service{
		ID:              didURI,
		Type:            "IndyAgent",
		Priority:        0,
		RecipientKeys:   []string{did.VerKey()},
		ServiceEndpoint: ae.Endp,
	}
	return &Doc{
		Context:   "https://w3id.org/did/v1",
		ID:        didURI,
		PublicKey: []PublicKey{pubK},
		Service:   []Service{service},
		Authentication: []VerificationMethod{{
			Type:      "Ed25519SignatureAuthentication2018",
			PublicKey: didURIRef,
		}},
	}
}
