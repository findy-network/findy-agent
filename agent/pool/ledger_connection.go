package pool

import (
	"github.com/findy-network/findy-agent/agent/async"
	indypool "github.com/findy-network/findy-wrapper-go/pool"
)

var pool async.Future

// Open opens ledger connection first time called. After that returns previous
// handle without checking the pool name. If caller wants to reopen new pool it
// must call Close() first.
//
// Note! We could have unit tests working with out ledger by reserving certain
// ledger handle and name, but that should be done in the indy Go wrapper
func Open(name string) *async.Future {
	// if we have a pool already open nothing to do
	if Handle() > 0 {
		return &pool
	}
	pool.SetChan(indypool.OpenLedger(name))
	return &pool
}

func Close() {
	oldPool := Handle()
	if oldPool != 0 {
		pool.SetChan(indypool.CloseLedger(oldPool))
		pool.Int() // call to make this a blocking call
	}
}

func Handle() (v int) {
	if pool.IsEmpty() {
		return 0
	}
	return pool.Int()
}
