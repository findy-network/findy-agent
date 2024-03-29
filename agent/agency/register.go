package agency

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-common-go/backup"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

var (
	Register utils.Reg // stores Agents already on-boarded, has Email as key
	Ready    readyTracker

	lastBackup = time.Now()
)

type readyTracker struct {
	ready bool
	l     sync.RWMutex
}

func (r *readyTracker) IsReady() bool {
	r.l.RLock()
	defer r.l.RUnlock()
	ready := r.ready
	return ready
}

func (r *readyTracker) RegisteringComplete() {
	r.l.Lock()
	defer r.l.Unlock()
	r.ready = true
}

func init() {
	err := Register.Load("")
	if err != nil {
		log.Panicln("Cannot load Agent registry:", err)
	}
}

// SaveRegistered saves registered CAs to the file. In most cases we handle this
// inside the package. This file based persistence system will change in the
// future.
func SaveRegistered() {
	err := Register.Save(utils.Settings.RegisterName())
	if err != nil {
		glog.Error(err)
		return
	}
	if timeToBackup() {
		// We leave file level sync handling for the OS for performance sake.
		// We could add lambda to Register.Save function to perform backup in
		// the same critical section as saving.
		go Backup()
	}
}

func timeToBackup() bool {
	interval := utils.Settings.RegisterBackupInterval()
	// optimize, if backup is not set
	if interval == 0 {
		return false
	}
	return time.Since(lastBackup) >= interval
}

func Backup() {
	defer err2.Catch(err2.Err(func(err error) {
		glog.Warning(err)
	}))

	backupFileName := utils.Settings.RegisterBackupName()
	if backupFileName == "" {
		glog.V(10).Infoln("register backup name is empty, will not backup")
		return
	}

	name := backupName(backupFileName)
	try.To(backup.FileCopy(utils.Settings.RegisterName(), name))
	glog.V(1).Infoln("CA register backup successful to:", name)
	lastBackup = time.Now()
}

// ResetRegistered sets the correct filename for our persistent storage and
// cleans it empty.
func ResetRegistered(filename string) error {
	utils.Settings.SetRegisterName(filename)
	fmt.Println("Note! Resetting handshake register, on-boarding starts over.")
	Ready.RegisteringComplete()
	return Register.Reset(filename)
}

func backupName(baseName string) string {
	tsStr := time.Now().Format(time.RFC3339)
	return backup.PrefixName(tsStr, baseName)
}
