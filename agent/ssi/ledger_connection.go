package ssi

import (
	indypool "github.com/optechlab/findy-go/pool"
)

var pool Future

// Open opens ledger connection first time called. After that returns previous
// handle without checking the pool name. If caller wants to reopen new pool it
// must call ClosePool() first.
//
// Note! We could have unit tests working with out ledger by reserving certain
// ledger handle and name, but that should be done in the indy Go wrapper
func OpenPool(name string) *Future {
	// if we have a pool already open nothing to do
	if Pool() > 0 {
		return &pool
	}
	pool.SetChan(indypool.OpenLedger(name))
	return &pool
}

func ClosePool() {
	oldPool := Pool()
	if oldPool != 0 {
		pool.SetChan(indypool.CloseLedger(oldPool))
		pool.Int() // call to make this a blocking call
	}
	return
}

func Pool() (v int) {
	if pool.IsEmpty() {
		return 0
	}
	return pool.Int()
}
