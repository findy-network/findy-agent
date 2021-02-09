package agency

import (
	"fmt"
	"log"
	"time"

	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-grpc/backup"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

var (
	Register utils.Reg // stores Agents already on-boarded, has Email as key

	lastBackup = time.Now()
)

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
	return time.Now().Sub(lastBackup) >= interval
}

func Backup() {
	defer err2.CatchTrace(func(err error) {
		glog.Warning(err)
	})

	name := backupName(utils.Settings.RegisterBackupName())
	err2.Check(backup.FileCopy(utils.Settings.RegisterName(), name))
	glog.V(1).Infoln("CA register backup successful to:", name)
	lastBackup = time.Now()
}

// ResetRegistered sets the correct filename for our persistent storage and
// cleans it empty.
func ResetRegistered(filename string) error {
	utils.Settings.SetRegisterName(filename)
	fmt.Println("Note! Resetting handshake register, on-boarding starts over.")
	return Register.Reset(filename)
}

func backupName(baseName string) string {
	tsStr := time.Now().Format(time.RFC3339)
	return backup.PrefixName(tsStr, baseName)
}
