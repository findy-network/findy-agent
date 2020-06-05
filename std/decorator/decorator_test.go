package decorator

import (
	"reflect"
	"testing"
)

func TestNewThread(t *testing.T) {
	type args struct {
		ID  string
		PID string
	}
	tests := []struct {
		name string
		args args
		want *Thread
	}{
		{"PID empty", args{ID: "12345", PID: ""}, &Thread{ID: "12345"}},
		{"PID same", args{ID: "12345", PID: "12345"}, &Thread{ID: "12345"}},
		{"PID different", args{ID: "12345", PID: "123456"}, &Thread{ID: "12345", PID: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewThread(tt.args.ID, tt.args.PID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewThread() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckThread(t *testing.T) {
	orgID := "ORG_ID_VALUE"
	id := "ID_VALUE"
	pid := "PID_VALUE"
	want := &Thread{ID: id}
	wantOrg := &Thread{ID: orgID}
	wantPID := &Thread{ID: id, PID: pid}
	wantOrgWithPID := &Thread{ID: orgID, PID: pid}

	type args struct {
		thread *Thread
		ID     string
	}
	tests := []struct {
		name string
		args args
		want *Thread
	}{
		{"was nil", args{thread: nil, ID: id}, want},
		{"was empty", args{thread: &Thread{}, ID: id}, want},
		{"was pid", args{thread: &Thread{ID: "", PID: pid}, ID: id}, wantPID},
		{"was org", args{thread: &Thread{ID: orgID}, ID: id}, wantOrg},
		{"was org and pid", args{thread: &Thread{ID: orgID, PID: pid}, ID: id}, wantOrgWithPID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckThread(tt.args.thread, tt.args.ID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckThread() = %v, want %v", got, tt.want)
			}
		})
	}
}
