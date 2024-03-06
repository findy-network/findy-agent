package ssi

import (
	"reflect"
	"testing"
)

func TestCache_lazyAdd(t *testing.T) {
	type fields struct {
		cache map[string]*DID
	}
	type args struct {
		s string
		d *DID
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"1st", fields{nil}, args{"DID_STRING", NewDid("DID_STRING", "VER_KEY")}},
		{"2nd", fields{nil}, args{"DID_STRING2", NewDid("DID_STRING2", "VER_KEY2")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			c := &Cache{
				cache: tt.fields.cache,
			}
			c.LazyAdd(tt.args.s, tt.args.d)
		})
	}
}

func TestCache_get(t *testing.T) {
	c := Cache{}
	c.Add(NewDid("DID_STRING", "VER_KEY"))
	c.Add(NewDid("DID_STRING1", "VER_KEY1"))
	c.Add(NewDid("DID_STRING2", "VER_KEY2"))

	type fields struct {
		cache map[string]*DID
	}
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *DID
	}{
		{"1", fields{c.cache}, args{"DID_STRING"}, NewDid("DID_STRING", "VER_KEY")},
		{"2", fields{c.cache}, args{"DID_STRING1"}, NewDid("DID_STRING1", "VER_KEY1")},
		{"3", fields{c.cache}, args{"DID_STRING2"}, NewDid("DID_STRING2", "VER_KEY2")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cache{
				cache: tt.fields.cache,
			}
			if got := c.Get(tt.args.s, false); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cache.get() = %v, want %v", got, tt.want)
			}
		})
	}
}
