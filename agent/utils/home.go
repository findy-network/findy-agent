package utils

import (
	"os"
	"os/user"
)

// IndyBaseDir TODO: the function name is bad, why I cannot remember?
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
