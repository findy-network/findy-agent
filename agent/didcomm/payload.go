/*
Package didcomm is package to offer interfaces for all types of the didcomm
messages. The package helps to abstract indy's legacy messages as well as new
Aries messages. Corresponding packages are mesg and aries.

The package offers needed interfaces, some helper functions and variables. More
information can be found from each individual type.
*/
package didcomm

import (
	"strings"

	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/core"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/golang/glog"
)

// CreatorGod creates payloads and messages by namespace string.
var CreatorGod = God{}

type God struct {
	plCreators  map[string]PayloadFactor
	msgCreators map[string]MsgFactor
}

func (g *God) AddPayloadCreator(namespace string, c PayloadFactor) {
	if g.plCreators == nil {
		g.plCreators = make(map[string]PayloadFactor)
	}
	g.plCreators[namespace] = c
}

func (g *God) AddMsgCreator(namespace string, c MsgFactor) {
	if g.msgCreators == nil {
		g.msgCreators = make(map[string]MsgFactor)
	}
	g.msgCreators[namespace] = c
}

func (g *God) PayloadCreator(namespace string) PayloadFactor {
	return g.plCreators[namespace]
}

func (g *God) MsgCreator(namespace string) MsgFactor {
	return g.msgCreators[namespace]
}

func (g *God) MsgCreatorByType(t string) MsgFactor {
	return g.msgCreators[FieldAtInd(t, 0)]
}

func (g *God) PayloadCreatorByType(t string) PayloadFactor {
	return g.plCreators[FieldAtInd(t, 0)]
}

type PayloadHdr interface {
	ID() string
	Type() string
}

type PayloadWriteHdr interface {
	SetID(id string)
	SetType(t string)
}

type PayloadThread interface {
	Thread() *decorator.Thread
	SetThread(t *decorator.Thread)
	ThreadID() string
}

type JSONSpeaker interface {
	JSON() []byte
}

type Payload interface {
	PayloadHdr
	PayloadThread
	JSONSpeaker

	SetType(t string)

	Creator() PayloadFactor
	MsgCreator() MsgFactor

	FieldObj() interface{}

	Message() Msg       // this is mostly for legacy message handling, before Aries
	MsgHdr() MessageHdr // this generic and preferable way to get the message

	Protocol() string
	ProtocolMsg() string
	Namespace() string
}

type Factor interface {
	NewMessage(data []byte) MessageHdr
	NewMsg(init MsgInit) MessageHdr
}

type PayloadFactor interface {
	NewFromData(data []byte) Payload
	New(pi PayloadInit) Payload
	NewMsg(id, t string, m MessageHdr) Payload
}

type PayloadInit struct {
	ID   string
	Type string
	MsgInit
}

// MessageHdr is the base interface for all protocol messages. It has the
// minimum needed to handle and process inbound and outbound protocol messages.
// The are message factors to help creation of these messages as well. This is
// the interface which should be used for all common references to didcomm
// messages. Please be noted that there still is a Payload concept which is a
// envelope level abstraction, and works well with the message.
type MessageHdr interface {
	PayloadHdr
	PayloadWriteHdr

	JSON() []byte
	Thread() *decorator.Thread
	FieldObj() interface{}
}

// CredentialAttribute for credential value
type CredentialAttribute struct {
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	MimeType string `json:"mime-type,omitempty"`
}

// ProofAttribute for proof request attributes
type ProofAttribute struct {
	ID        string `json:"-"`
	Name      string `json:"name,omitempty"`
	CredDefID string `json:"credDefId,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Value     string `json:"-"`
}

// ProofPredicate for proof request predicates
type ProofPredicate struct {
	ID     string `json:"-"`
	Name   string `json:"name,omitempty"`
	PType  string `json:"p_type,omitempty"`
	PValue int64  `json:"p_value,omitempty"`
}

// ProofValue for proof values
type ProofValue struct {
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	CredDefID string `json:"credDefId,omitempty"`
	Predicate string `json:"predicate,omitempty"`
}

// Msg is a legacy interface for before Aries message protocols. For new Aries
// protocols it isn't recommended to use it, but use MessageHdr instead.
type Msg interface {
	MessageHdr

	SubLevelID() string

	Ready() bool
}

// MsgInit is a helper struct for factors to construct new message instances.
type MsgInit struct {
	AID        string
	Type       string
	Nonce      string
	Error      string
	Did        string
	VerKey     string
	Endpoint   string
	EndpVerKey string
	RcvrEndp   service.Addr
	Name       string
	Info       string
	ID         string
	Ready      bool
	Thread     *decorator.Thread
	DIDObj     core.DID
	To         string
	Msg        map[string]interface{}
}

type MsgFactor interface {
	Factor

	Create(mcd MsgInit) MessageHdr
}

func FieldAtInd(s string, where int) string {
	if s == "" {
		return ""
	}

	maxSlits := 4
	if strings.HasPrefix(s, "https://") {
		maxSlits += 2
		where += 2
	}
	parts := strings.Split(s, "/")
	if len(parts) != maxSlits {
		glog.Error(s)
		panic("type string is not valid")
	}
	return parts[where]
}
