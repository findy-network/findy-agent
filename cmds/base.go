package cmds

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/lainio/err2"
	"github.com/optechlab/findy-agent/agent/cloud"
	"github.com/optechlab/findy-agent/agent/comm"
	"github.com/optechlab/findy-agent/agent/didcomm"
	"github.com/optechlab/findy-agent/agent/e2"
	"github.com/optechlab/findy-agent/agent/endp"
	"github.com/optechlab/findy-agent/agent/mesg"
	"github.com/optechlab/findy-agent/agent/pltype"
	"github.com/optechlab/findy-agent/agent/ssi"
	"github.com/optechlab/findy-agent/agent/utils"
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
	return ValidateKey(c.WalletKey)
}

func (c Cmd) ValidateWalletExistence(should bool) error {
	exists := ssi.NewRawWalletCfg(c.WalletName, c.WalletKey).Exists(false)
	ok := (should && exists) || (!should && !exists)
	if !ok {
		return fmt.Errorf("wallet exists: %v", exists)
	}
	return nil
}

func ValidateKey(k string) error {
	if k == "" {
		return errors.New("wallet key cannot be empty")
	}
	if len(k) != walletKeyLength {
		return errors.New("wallet key is not valid")
	}
	return nil
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
}

func NewCmd(d []byte) (c *Cmd, err error) {
	cmd := Cmd{}
	err = json.Unmarshal(d, &cmd)
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (c Cmd) Exec(p io.Writer, t string, m *mesg.Msg,
	f func(e Edge, im *mesg.Msg) (Result, error)) (r Result, err error) {

	defer err2.Return(&err)

	wcfg := ssi.NewRawWalletCfg(c.WalletName, c.WalletKey)
	edge := Edge{
		Cmd:   c,
		Agent: cloud.NewTransportReadyEA(wcfg),
	}
	defer edge.Agent.CloseWallet()

	im := e2.Msg.Try(edge.MsgToCA(p, t, m))

	return f(edge, &im)
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

// MsgToCA sends what ever message to an agency.
func (edge Edge) MsgToCA(w io.Writer,
	t string, msg *mesg.Msg) (in mesg.Msg, err error) {

	if edge.Agent == nil {
		edge.Agent = cloud.NewTransportReadyEA(
			ssi.NewRawWalletCfg(edge.WalletName, edge.WalletKey))
		defer func() { edge.Agent.CloseWallet() }()
	}

	trans := edge.Agent.Trans()
	if trans.PayloadPipe().IsNull() || trans.MessagePipe().IsNull() {
		return mesg.Msg{}, fmt.Errorf("wallet (%s) has no connection",
			edge.WalletName)
	}
	ipl, err := trans.Call(t, msg)
	if err != nil {
		return mesg.Msg{}, err
	}
	if ipl.Type == pltype.ConnectionError {
		return mesg.Msg{}, fmt.Errorf("%v", ipl.Message.Error)
	}
	return ipl.Message, nil
}

func Progress(w io.Writer) chan<- struct{} {
	done := make(chan struct{})
	go func() {
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
