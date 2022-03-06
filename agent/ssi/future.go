package ssi

import (
	"sync"

	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/lainio/err2/try"
)

type State uint32

const (
	empty State = iota
	triggered
	Consumed
)

type Future struct {
	On State
	V  interface{}
	ch findy.Channel
	lo sync.Mutex
}

// value returns actual result object from findy.Channel. It throws an err2
// exception if error happens.
func (f *Future) value() interface{} {
	f.lo.Lock()
	defer f.lo.Unlock()
	if f.On == triggered {
		r := <-f.ch
		f.On = Consumed
		f.V = r
		try.To(r.Err())
	}

	return f.V
}

func (f *Future) IsEmpty() bool {
	f.lo.Lock()
	defer f.lo.Unlock()
	return f.On == empty
}

func (f *Future) Result() (dtoResult *dto.Result) {
	pseudo := f.value()
	if pseudo != nil {
		r := pseudo.(dto.Result)
		dtoResult = &r
	}
	return
}

// NewFuture changes the existing findy.Channel to a Future.
func NewFuture(ch findy.Channel) *Future {
	f := &Future{}
	f.SetChan(ch)
	return f
}

// SetChan sets the existing findy.Channel to this Future.
func (f *Future) SetChan(ch findy.Channel) {
	f.lo.Lock()
	defer f.lo.Unlock()
	if f.On == triggered {
		// we have previous uneaten channel data, eat it off
		// this might be unnecessary because of garbage collector
		_ = f.value()
		// now it's empty and no one will ever know it, but we put this here
		// for to make semantics clear
		f.On = empty
	}
	f.ch = ch
	f.On = triggered
}

// MARK: type helpers for convenience, you could Result().GetHandle() for example.
//  now we have places for default values per type etc.

func (f *Future) Int() (i int) {
	r := f.Result()
	if r != nil {
		i = r.Handle()
	}
	return
}

func (f *Future) Strs() (s1, s2, s3 string) {
	r := f.Result()
	if r != nil {
		s1 = r.Str1()
		s2 = r.Str2()
		s3 = r.Str3()
	}
	return
}

func (f *Future) Bytes() (b []byte) {
	r := f.Result()
	if r != nil {
		b = r.Bytes()
	}
	return
}

func (f *Future) Str1() string {
	str1, _, _ := f.Strs()
	return str1
}

func (f *Future) Str2() string {
	_, str2, _ := f.Strs()
	return str2
}
