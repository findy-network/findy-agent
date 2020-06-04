package agency

import (
	"fmt"
	"log"

	"github.com/golang/glog"
	"github.com/findy-network/findy-agent/agent/utils"
)

var Register utils.Reg // stores Agents already on-boarded, has Email as key

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
	}
}

// ResetRegistered sets the correct filename for our persistent storage and
// cleans it empty.
func ResetRegistered(filename string) error {
	utils.Settings.SetRegisterName(filename)
	fmt.Println("Note! Resetting handshake register, on-boarding starts over.")
	return Register.Reset(filename)
}
