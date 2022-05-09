package method

import (
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
	return s[1]
}

func DIDType(s string) Type {
	t, ok := methodTypes[String(s)]
	if !ok {
		glog.Warningf("cannot compute did method from '%s'", s)
		return TypeUnknown
	}
	return t
}

func Accept(did core.DID, t Type) bool {
	return DIDType(did.String()) == t
}

var methodTypes = map[string]Type{
	"unknown": TypeUnknown,
	"key":     TypeKey,
	"peer":    TypePeer,
	"sov":     TypeSov,
	"indy":    TypeIndy,
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
		assert.D.Truef(false, "did method (%v) not supported", method)
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
	defer err2.Return(&err)

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
