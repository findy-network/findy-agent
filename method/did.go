package method

import (
	"fmt"
	"strings"
	"sync"

	"github.com/findy-network/findy-agent/agent/managed"
	"github.com/findy-network/findy-agent/core"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

func String(d string) string {
	s := strings.Split(d, ":")
	if len(s) == 1 && s[0] == d {
		// we have legacy DID format, i.e. no prefix
		return unknownTypeStr
	}
	return s[1]
}

func DIDType(s string) Type {
	if s == "" {
		return TypeUnknown
	}

	t, ok := methodTypes[String(s)]
	if !ok {
		glog.Warningf("cannot compute did method from '%s'", s)
		return TypeUnknown
	}
	return t
}

func Accept(did core.DID, t Type) bool {
	return DIDType(did.URI()) == t
}

const unknownTypeStr = "unknown"

var methodTypes = map[string]Type{
	unknownTypeStr: TypeUnknown,

	"key":  TypeKey,
	"peer": TypePeer,
	"sov":  TypeSov,
	"indy": TypeIndy,
}

type Type int

const (
	TypeUnknown Type = 0 + iota
	TypeKey
	TypePeer
	TypeSov
	TypeIndy
)

func (t Type) String() string {
	return []string{"unknown", "key", "peer", "sov", "indy"}[t]
}

func (t Type) DIDString() string {
	return fmt.Sprintf("did:%s:", t.String())
}

func New(
	method Type,
	hStorage managed.Wallet,
	args ...string,
) (
	id core.DID,
	err error,
) {
	switch method {
	case TypePeer:
		return NewPeer(hStorage, args...)
	case TypeKey:
		return NewKey(hStorage, args...)
	default:
		assert.That(false, "did method (%v) not supported", method)
	}
	return
}

func NewFromDID(
	hStorage managed.Wallet,
	didStr ...string,
) (
	id core.DID,
	err error,
) {
	defer err2.Handle(&err)

	switch DIDType(didStr[0]) {
	case TypePeer:
		assert.SLen(didStr, 2)
		return NewPeerFromDoc(hStorage, didStr[1])
	case TypeKey:
		assert.SLen(didStr, 1)
		return NewKeyFromDID(hStorage, didStr[0])
	default:
		assert.NotImplemented()
	}
	return
}

var _ = struct {
	sync.Mutex
	dids map[string]Peer
}{
	dids: make(map[string]Peer),
}
