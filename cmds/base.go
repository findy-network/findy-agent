package cmds

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

const walletKeyLength1 = 43
const walletKeyLength2 = 44

var ErrInvalid = errors.New("invalid command, check arguments")

type Cmd struct {
	WalletName string `cmd_usage:"wallet name is required"`
	WalletKey  string `cmd_usage:"wallet key is required"`
}

func (c Cmd) Validate() error {
	if c.WalletName == "" {
		return errors.New("wallet name cannot be empty")
	}
	return c.ValidateWalletKey()
}

func (c Cmd) ValidateWalletKey() error {
	return ValidateKey(c.WalletKey, "wallet")
}

func (c Cmd) ValidateWalletExistence(should bool) error {
	exists := ssi.NewRawWalletCfg(c.WalletName, c.WalletKey).Exists()
	ok := (should && exists) || (!should && !exists)
	if !ok {
		return fmt.Errorf("wallet exists: %v", exists)
	}
	return nil
}

func ValidateKey(k string, name string) error {
	if k == "" {
		return fmt.Errorf("%s key cannot be empty", name)
	}
	if len(k) != walletKeyLength1 && len(k) != walletKeyLength2 {
		return fmt.Errorf("%s key is not valid %d <> %d|%d", name,
			len(k), walletKeyLength1, walletKeyLength2)
	}
	return nil
}

func ValidateTime(t string) error {
	if !timeFmtRegExp.MatchString(t) {
		return errors.New("time format mismatch (HH:MM or HH:MM:SS)")
	}
	return nil
}

var timeFmtRegExp = regexp.MustCompile(`^([0-1]?\d|2[0-3]):([0-5]?\d)(?::([0-5]?\d))?$`)

func ValidateSeed(seed string) error {
	if seed != "" && len(seed) != 32 {
		return errors.New("seed must be empty or length of 32")
	}
	return nil
}

type Edge struct {
	Cmd
	*cloud.Agent
}

type Result interface {
	JSON() ([]byte, error)
}

type Command interface {
	Validate() error
	Exec(w io.Writer) (r Result, err error)
	RPCExec(w io.Writer) (r Result, err error)
}

func NewCmd(d []byte) (c *Cmd, err error) {
	cmd := Cmd{}
	err = json.Unmarshal(d, &cmd)
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// Fprintln is fmt.Fprintln but it allows writer to be nil. Note! it throws an
// error.
func Fprintln(w io.Writer, a ...interface{}) {
	if w != nil {
		try.To1(fmt.Fprintln(w, a...))
	}
}

// Fprintf is fmt.Fprintf but it allows writer to be nil. Note! it throws an
// error.
func Fprintf(w io.Writer, format string, a ...interface{}) {
	if w != nil {
		try.To1(fmt.Fprintf(w, format, a...))
	}
}

// Fprintf is fmt.Fprintf but it allows writer to be nil. Note! it throws an
// error.
func Fprint(w io.Writer, a ...interface{}) {
	if w != nil {
		try.To1(fmt.Fprint(w, a...))
	}
}

func Progress(w io.Writer) chan<- struct{} {
	done := make(chan struct{})
	go func() {
		defer err2.Catch(func(err error) {
			glog.Error(err)
		})
		for {
			select {
			case <-done:
				return
			case <-time.After(300 * time.Millisecond):
				Fprint(w, ".")
			}
		}
	}()
	return done
}
