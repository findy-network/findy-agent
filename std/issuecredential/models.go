/*
Taken from aries-framework-go, and heavily modified. The idea is to replace
these with the aries-framework-go when it's ready. Until now we use our own
minimalistic solution.

Most important modification were 1) renaming structures: removing Credential
word which is already in the package name, and 2) adding thread decorators to
all, and 3) IDs.
*/

// Package issuecredential is package for Aries protocol messages for same name.
package issuecredential

import "github.com/optechlab/findy-agent/std/decorator"

// Propose is an optional message sent by the potential Holder to the Issuer
// to initiate the protocol or in response to a offer-credential message when the Holder
// wants some adjustments made to the credential data offered by Issuer.
type Propose struct {
	ID   string `json:"@id,omitempty"`
	Type string `json:"@type,omitempty"`
	// Comment is an optional field that provides human readable information about this Credential Offer,
	// so the offer can be evaluated by human judgment.
	// TODO: Should follow DIDComm conventions for l10n. [Issue #1300]
	Comment string `json:"comment,omitempty"`
	// CredentialProposal is an optional JSON-LD object that represents
	// the credential data that the Prover wants to receive.
	CredentialProposal PreviewCredential `json:"credential_proposal,omitempty"`
	// SchemaIssuerDid is an optional filter to request credential based on a particular Schema issuer DID.
	SchemaIssuerDid string `json:"schema_issuer_did,omitempty"`
	// SchemaID is an optional filter to request credential based on a particular Schema.
	// This might be helpful when requesting a version 1 passport instead of a version 2 passport, for example.
	SchemaID string `json:"schema_id,omitempty"`
	// SchemaName is an optional filter to request credential based on a schema name.
	// This is useful to allow a more user-friendly experience of requesting a credential by schema name.
	SchemaName string `json:"schema_name,omitempty"`
	// SchemaVersion is an optional filter to request credential based on a schema version.
	// This is useful to allow a more user-friendly experience of requesting a credential by schema name and version.
	SchemaVersion string `json:"schema_version,omitempty"`
	// CredDefID is an optional filter to request credential based on a particular Credential Definition.
	// This might be helpful when requesting a commercial driver's license instead of
	// an ordinary driver's license, for example.
	CredDefID string `json:"cred_def_id,omitempty"`
	// IssuerDid is an optional filter to request a credential issued by the owner of a particular DID.
	IssuerDid string `json:"issuer_did,omitempty"`

	Thread *decorator.Thread `json:"~thread,omitempty"`
}

// Offer is a message sent by the Issuer to the potential Holder,
// describing the credential they intend to offer and possibly the price they expect to be paid.
// TODO: Need to add ~payment_request and ~timing.expires_time decorators [Issue #1297]
type Offer struct {
	ID   string `json:"@id,omitempty"`
	Type string `json:"@type,omitempty"`
	// Comment is an optional field that provides human readable information about this Credential Offer,
	// so the offer can be evaluated by human judgment.
	// TODO: Should follow DIDComm conventions for l10n. [Issue #1300]
	Comment string `json:"comment,omitempty"`
	// CredentialPreview is a JSON-LD object that represents the credential data that Issuer is willing to issue.
	CredentialPreview PreviewCredential `json:"credential_preview,omitempty"`
	// OffersAttach is a slice of attachments that further define the credential being offered.
	// This might be used to clarify which formats or format versions will be issued.
	OffersAttach []decorator.Attachment `json:"offers~attach,omitempty"`

	Thread *decorator.Thread `json:"~thread,omitempty"`
}

// Request is a message sent by the potential Holder to the Issuer,
// to request the issuance of a credential. Where circumstances do not require
// a preceding Offer Credential message (e.g., there is no cost to issuance
// that the Issuer needs to explain in advance, and there is no need for cryptographic negotiation),
// this message initiates the protocol.
// TODO: Need to add ~payment-receipt decorator [Issue #1298]
type Request struct {
	ID   string `json:"@id,omitempty"`
	Type string `json:"@type,omitempty"`
	// Comment is an optional field that provides human readable information about this Credential Offer,
	// so the offer can be evaluated by human judgment.
	// TODO: Should follow DIDComm conventions for l10n. [Issue #1300]
	Comment string `json:"comment,omitempty"`
	// RequestsAttach is a slice of attachments defining the requested formats for the credential
	RequestsAttach []decorator.Attachment `json:"requests~attach,omitempty"`

	Thread *decorator.Thread `json:"~thread,omitempty"`
}

// Issue contains as attached payload the credentials being issued and is
// sent in response to a valid Request Credential message.
// TODO: Need to add ~please-ack decorator [Issue #1299]
type Issue struct {
	ID   string `json:"@id,omitempty"`
	Type string `json:"@type,omitempty"`
	// Comment is an optional field that provides human readable information about this Credential Offer,
	// so the offer can be evaluated by human judgment.
	// TODO: Should follow DIDComm conventions for l10n. [Issue #1300]
	Comment string `json:"comment,omitempty"`
	// CredentialsAttach is a slice of attachments containing the issued credentials.
	CredentialsAttach []decorator.Attachment `json:"credentials~attach,omitempty"`

	Thread *decorator.Thread `json:"~thread,omitempty"`
}

// Preview is used to construct a preview of the data for the credential that is to be issued.
type PreviewCredential struct {
	Type       string      `json:"@type,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
}

// Attribute describes an attribute for a Preview Credential
type Attribute struct {
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime-type,omitempty"`
	Value    string `json:"value,omitempty"`
}
