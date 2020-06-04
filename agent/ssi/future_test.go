package ssi

import (
	"reflect"
	"testing"

	"github.com/lainio/err2"
	"github.com/optechlab/findy-go"
	"github.com/optechlab/findy-go/dto"
)

func fillChannel(cmd uint32, ch findy.Channel) {
	r := dto.Result{}
	r.SetHandle(int(cmd))
	//	time.Sleep(time.Duration(cmd) * 10 * time.Millisecond)
	ch <- r
}

func fillChannelWithError(ch findy.Channel) {
	r := dto.Result{
		Er: dto.Err{
			Error: "TEST_ERROR",
			Code:  100,
		},
	}
	ch <- r
}

func TestFuture_GetValue_And_SetChan(t *testing.T) {
	const HandleValueToTest = 1
	result := dto.Result{Data: dto.Data{Handle: HandleValueToTest}}

	myFuture := Future{}
	ch := make(findy.Channel, 1)

	tests := []struct {
		name string
		want interface{}
	}{
		{"1st", nil},
		{"2nd", result},
		{"3rd", result},
		{"4th", result},
		{"5th", result},
		{"6th", result},
		{"7th", result},
		{"8th", result},
		{"9th", result},
	}
	for i, tt := range tests {
		if i == 1 || i == 3 || i == 6 { // write value to channel
			myFuture.SetChan(ch)               // order of these lines are..
			fillChannel(HandleValueToTest, ch) // not important
		}
		t.Run(tt.name, func(t *testing.T) {
			f := &myFuture
			if got := f.value(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Future.value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuture_TestWithErrorResult(t *testing.T) {
	readValue := func(f *Future) {
		defer err2.Catch(func(err error) {
			// eat the error
		})
		f.value()
	}

	result := dto.Result{
		Er: dto.Err{
			Error: "TEST_ERROR",
			Code:  100,
		},
	}

	myFuture := Future{}
	ch := make(findy.Channel, 1)

	tests := []struct {
		name string
		want interface{}
	}{
		{"1st", result},
		{"2nd", result},
		{"3rd", result},
	}
	for i, tt := range tests {
		if i == 0 {
			myFuture.SetChan(ch)
			fillChannelWithError(ch)
			readValue(&myFuture)
		}
		t.Run(tt.name, func(t *testing.T) {
			if got := myFuture.value(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Future.value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuture_GetResult(t *testing.T) {
	result := dto.Result{Data: dto.Data{Handle: 1}}
	result2 := dto.Result{Data: dto.Data{Handle: 1}}
	type fields struct {
		v  interface{}
		ch findy.Channel
	}
	tests := []struct {
		name          string
		fields        fields
		wantDtoResult *dto.Result
	}{
		{"1st", fields{v: result}, &result},
		{"2nd", fields{v: result2}, &result2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Future{
				V:  tt.fields.v,
				ch: tt.fields.ch,
			}
			if gotDtoResult := f.Result(); !reflect.DeepEqual(gotDtoResult, tt.wantDtoResult) {
				t.Errorf("Future.Result() = %v, want %v", gotDtoResult, tt.wantDtoResult)
			}
		})
	}
}

func TestFuture_GetInt(t *testing.T) {
	result := dto.Result{}
	result1 := dto.Result{Data: dto.Data{Handle: 1}}
	result2 := dto.Result{Data: dto.Data{Handle: 2}}

	type fields struct {
		v  interface{}
		ch findy.Channel
	}
	tests := []struct {
		name   string
		fields fields
		wantI  int
	}{
		{"zero", fields{v: result}, 0},
		{"1st", fields{v: result1}, 1},
		{"2nd", fields{v: result2}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Future{
				V:  tt.fields.v,
				ch: tt.fields.ch,
			}
			if gotI := f.Int(); gotI != tt.wantI {
				t.Errorf("Future.Int() = %v, want %v", gotI, tt.wantI)
			}
		})
	}
}

func TestFuture_GetStrs(t *testing.T) {
	result := dto.Result{}
	result1 := dto.Result{Data: dto.Data{Str1: "str1", Str2: "str2", Str3: ""}}
	result2 := dto.Result{Data: dto.Data{Str1: "str1", Str2: "str2", Str3: "str3"}}

	type fields struct {
		v  interface{}
		ch findy.Channel
	}
	tests := []struct {
		name   string
		fields fields
		wantS1 string
		wantS2 string
		wantS3 string
	}{
		{"zero", fields{v: result}, "", "", ""},
		{"1st", fields{v: result1}, "str1", "str2", ""},
		{"2nd", fields{v: result2}, "str1", "str2", "str3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Future{
				V:  tt.fields.v,
				ch: tt.fields.ch,
			}
			gotS1, gotS2, gotS3 := f.Strs()
			if gotS1 != tt.wantS1 {
				t.Errorf("Future.Strs() gotS1 = %v, want %v", gotS1, tt.wantS1)
			}
			if gotS2 != tt.wantS2 {
				t.Errorf("Future.Strs() gotS2 = %v, want %v", gotS2, tt.wantS2)
			}
			if gotS3 != tt.wantS3 {
				t.Errorf("Future.Strs() gotS3 = %v, want %v", gotS3, tt.wantS3)
			}
		})
	}
}
