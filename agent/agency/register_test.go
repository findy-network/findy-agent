package agency

import (
	"testing"
	"time"

	"github.com/findy-network/findy-agent/agent/utils"
)

func Test_timeToBackup(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		sleep    time.Duration
		want     bool
	}{
		{name: "not yet", interval: time.Hour, sleep: 0, want: false},
		{name: "zero interval means no backup", interval: 0, sleep: time.Millisecond, want: false},
		{name: "zero interval means no backup 2nd", interval: 0, sleep: 2 * time.Millisecond, want: false},
		{name: "2 milli interval sleep 1", interval: 2 * time.Millisecond, sleep: time.Millisecond, want: false},
		{name: "1 milli interval sleep 2", interval: time.Millisecond, sleep: 2 * time.Millisecond, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastBackup = time.Now()
			utils.Settings.SetRegisterBackupInterval(tt.interval)
			time.Sleep(tt.sleep)
			if got := timeToBackup(); got != tt.want {
				t.Errorf("timeToBackup() = %v, want %v", got, tt.want)
			}
		})
	}
}
