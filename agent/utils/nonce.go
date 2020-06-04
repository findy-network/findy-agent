package utils

import (
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
	"sync"

	"github.com/golang/glog"
	"github.com/google/uuid"
)

var generator = &nonce{ns: make(map[uint64]struct{})}

type nonce struct {
	ns map[uint64]struct{}
	l  sync.Mutex
}

func (no *nonce) dispose(n uint64) {
	no.l.Lock()
	defer no.l.Unlock()
	delete(no.ns, n)
}

func (no *nonce) reserve(n uint64) {
	no.l.Lock()
	defer no.l.Unlock()
	no.ns[n] = struct{}{}
}

func gen() uint64 {
	m := big.NewInt(math.MaxInt64)
	r, err := rand.Int(rand.Reader, m)
	if err != nil {
		panic("cannot create nonce")
	}
	return r.Uint64()
}

// NewNonce generates new uint64 nonce with Go's crypto package
func NewNonce() uint64 {
	return gen()
}

// NewNonceStr generates new nonce with Go's crypto package, and returns value
// as string.
func NewNonceStr() string {
	return NonceToStr(NewNonce())
}

// UUID generates new nonce with Go's crypto package, and returns value
// as string.
func UUID() string {
	return uuid.New().String()
}

// DisposeNonce frees the nonce value for others to use.
func DisposeNonce(n uint64) {
	generator.dispose(n)
}

func ReserveNonce(n uint64) uint64 {
	generator.reserve(n)
	return n
}

func NonceToStr(n uint64) string {
	s := strconv.FormatUint(n, 10)
	return s
}

func NonceNum(s string) uint64 {
	sn := s
	if sn == "" {
		sn = "0"
	}
	n, err := strconv.ParseUint(sn, 10, 64)
	if err != nil {
		glog.Warning("Error nonce conversion! Using zero")
		n = 0
	}
	return n
}

func ParseNonce(ns string) uint64 {
	n, err := strconv.ParseInt(ns, 10, 64)
	nonce := uint64(n)
	if err != nil {
		// we use nonce like multipurpose, so this is newer critical
		//log.Println("Cannot parse nonce")
		nonce = 0
	}
	return nonce
}
