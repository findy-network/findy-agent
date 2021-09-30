package cmds

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/findy-network/findy-agent/agent/cloud"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

const walletKeyLength = 44

var ErrInvalid = errors.New("invalid command, check arguments")

type Cmd struct {
	WalletName string `cmd_usage:"wallet name is required"`
	WalletKey  string `cmd_usage:"wallet key is required"`
}

func (c Cmd) Validate() error {
	if c.WalletName == "" {
		return errors.New("wallet name cannot be empty")
	}
	if err := c.ValidateWalletKey(); err != nil {
		return err
	}
	return nil
}

func (c Cmd) ValidateWalletKey() error {
	return ValidateKey(c.WalletKey, "wallet")
}

func (c Cmd) ValidateWalletExistence(should bool) error {
	exists := ssi.NewRawWalletCfg(c.WalletName, c.WalletKey).Exists(false)
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
	if len(k) != walletKeyLength {
		return fmt.Errorf("%s key is not valid (%d/%d)", name,
			len(k), walletKeyLength)
	}
	return nil
}

func ValidateTime(t string) error {
	if !r.MatchString(t) {
		return errors.New("time format mismatch (HH:MM or HH:MM:SS)")
	}
	return nil
}

var r *regexp.Regexp

func init() {
	r, _ = regexp.Compile("^([0-1]?\\d|2[0-3]):([0-5]?\\d)(?::([0-5]?\\d))?$")
}

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
	RpcExec(w io.Writer) (r Result, err error)
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
		err2.Empty.Try(fmt.Fprintln(w, a...))
	}
}

// Fprintf is fmt.Fprintf but it allows writer to be nil. Note! it throws an
// error.
func Fprintf(w io.Writer, format string, a ...interface{}) {
	if w != nil {
		err2.Empty.Try(fmt.Fprintf(w, format, a...))
	}
}

// Fprintf is fmt.Fprintf but it allows writer to be nil. Note! it throws an
// error.
func Fprint(w io.Writer, a ...interface{}) {
	if w != nil {
		err2.Empty.Try(fmt.Fprint(w, a...))
	}
}

func Progress(w io.Writer) chan<- struct{} {
	done := make(chan struct{})
	go func() {
		defer err2.CatchTrace(func(err error) {
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

func SendHandshake(msg *mesg.Msg, endpoint *endp.Addr) (payload *mesg.Payload, err error) {
	p := mesg.Payload{ID: msg.Nonce, Type: pltype.ConnectionHandshake, Message: *msg}
	// BLOCKING CALL to make endpoint request this time, proper for handshakes
	return PostRequest(endpoint.Address(), bytes.NewReader(p.JSON()), utils.Settings.Timeout())
}

func PostRequest(urlStr string, msg io.Reader, timeout time.Duration) (p *mesg.Payload, err error) {
	data, err := comm.SendAndWaitReq(urlStr, msg, timeout)
	if err != nil {
		return nil, fmt.Errorf("reading body: %s", err)
	}
	p = mesg.NewPayload(data)
	if p.Message.Error != "" {
		err = fmt.Errorf("http POST response: %s", p.Message.Error)
	}

	return
}

func SendAndWaitDIDComPayload(p didcomm.Payload, endpoint *endp.Addr, nonce uint64) (rp didcomm.Payload, err error) {
	// BLOCKING CALL to make endpoint request this time, proper for handshakes
	pl, err := PostRequest(endpoint.Address(), bytes.NewReader(p.JSON()), utils.Settings.Timeout())
	return mesg.NewPayloadImpl(pl), err
}

func SendAndWaitPayload(p *mesg.Payload, endpoint *endp.Addr, nonce uint64) (rp *mesg.Payload, err error) {
	// BLOCKING CALL to make endpoint request this time, proper for handshakes
	return PostRequest(endpoint.Address(), bytes.NewReader(p.JSON()), utils.Settings.Timeout())
}
