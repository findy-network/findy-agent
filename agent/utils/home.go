package utils

import (
	"os"
	"os/user"
)

func IndyBaseDir() string {
	if v := os.Getenv("HOME"); v != "" {
		return v
	}
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	return currentUser.HomeDir
}
