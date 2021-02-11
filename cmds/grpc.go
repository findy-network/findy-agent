package cmds

import "errors"

type GrpcCmd struct {
	TlsPath string
	Addr    string
	Port    int
	AdminID string
}

func (c GrpcCmd) Validate() error {
	if c.Addr == "" {
		return errors.New("server address cannot be empty")
	}
	if c.AdminID == "" {
		return errors.New("server admin id cannot be empty")
	}
	if c.Port == 0 {
		return errors.New("server port cannot be zero")
	}
	return nil
}
