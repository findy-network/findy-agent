package trans

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/sec"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

// Transport is communication mechanism between EA to CA, client to server.
// Server side is not supported yet, but coming. With Transport EA can
// communicate easily with its CA.
type Transport struct {
	PLPipe  sec.Pipe // Payload communication pipe
	MsgPipe sec.Pipe // Message communication
	Endp    string   // Given endpoint
}

func (tr Transport) DIDComCallEndp(endp, msgType string, msg didcomm.Msg) (rp didcomm.Payload, err error) {
	if msg.Nonce() == "" || endp == "" {
		return nil,
			errors.New("cannot call endpoint with empty nonce or endp")
	}

	glog.V(5).Info(tr)

	var cryptMsg didcomm.Msg
	if msgType != pltype.ConnectionOffer && msgType != pltype.ConnectionHandshake {
		cryptMsg = tr.EncDIDComMsg(msg)
	} else {
		cryptMsg = msg
	}

	cg := didcomm.CreatorGod.PayloadCreatorByType(msgType)
	p := cg.NewMsg(tr.MsgPipe.In.Did(), msgType, cryptMsg)

	payload, err := tr.sendEndpDIDComPL(endp, p)
	return payload, err
}

func (tr Transport) String() string {
	inStr := tr.PLPipe.In.Did()
	outStr := tr.PLPipe.Out.Did()

	if inStr != tr.MsgPipe.In.Did() {
		inStr += "/" + tr.MsgPipe.In.Did()
	}

	inWallet := strconv.Itoa(tr.PLPipe.In.Wallet())
	if tr.PLPipe.In.Wallet() != tr.MsgPipe.In.Wallet() {
		inWallet += "/" + strconv.Itoa(tr.MsgPipe.In.Wallet())
	}

	if outStr != tr.MsgPipe.Out.Did() {
		outStr += "/" + tr.MsgPipe.Out.Did()
	}

	return fmt.Sprintf("In: %s(%s) Out: %s Ep: %s",
		inStr, inWallet, outStr, tr.Endp)
}

func (tr Transport) EndpAddr() string {
	return tr.Endp
}

func (tr Transport) PayloadPipe() sec.Pipe {
	return tr.PLPipe
}

func (tr Transport) MessagePipe() sec.Pipe {
	return tr.MsgPipe
}

func (tr Transport) Endpoint() string {
	return tr.Endp
}

func (tr Transport) SetMessageOut(d *ssi.DID) {
	tr.MsgPipe.Out = d
}

// cnxAddr returns *Addr for making HTTP requests. The *Addr is build
// from DID data and Endp string. The Endp can be empty which case
// Msg.Out.endpoint() is used to fetch endpoint data from wallet.
func (tr Transport) cnxAddr() *endp.Addr {
	endpoint := tr.Endp
	if endpoint == "" {
		endpoint = tr.MsgPipe.Out.Endpoint()
	}
	addr := endp.NewClientAddr(endpoint)
	addr.MsgRcvr = addr.PlRcvr
	addr.PlRcvr = tr.PLPipe.Out.Did()
	return addr
}

func (tr Transport) sendEndpDIDComPL(endp string, p didcomm.Payload) (didcomm.Payload, error) {
	data := tr.PLPipe.Encrypt(p.JSON())
	responseData, err := comm.SendAndWaitReq(endp,
		bytes.NewReader(data), utils.Settings.Timeout())
	if err != nil {
		return nil, fmt.Errorf("send to CA: %s", err)
	}
	payload := tr.receiveDIDComPayload(responseData)
	return payload, err
}

func (tr Transport) Call(msgType string, msg *mesg.Msg) (rp *mesg.Payload, err error) {
	sendMsg := *msg
	originalNonce := utils.NewNonceStr()
	if sendMsg.Nonce != "" {
		glog.Warning("nonce isn't empty:", sendMsg.Nonce)
	}
	sendMsg.Nonce = originalNonce

	var cryptMsg *mesg.Msg
	if msgType != pltype.ConnectionOffer && msgType != pltype.ConnectionHandshake {
		cryptMsg = tr.EncMsg(&sendMsg)
	} else {
		cryptMsg = &sendMsg
	}

	p := mesg.Payload{ID: tr.MsgPipe.In.Did(), Type: msgType, Message: *cryptMsg}

	n := utils.NonceNum(originalNonce)
	payload, err := tr.SendPayload(&p, n)
	return payload, err
}

func (tr Transport) SendPayload(p *mesg.Payload, orgNonce uint64) (*mesg.Payload, error) {
	data := tr.PLPipe.Encrypt(dto.ToJSONBytes(p))
	responseData, err := comm.SendAndWaitReq(tr.cnxAddr().Address(),
		bytes.NewReader(data), utils.Settings.Timeout())
	if err != nil {
		return nil, fmt.Errorf("send to CA: %s", err)
	}
	payload := tr.receivePayload(responseData, orgNonce)
	return payload, err
}

func (tr Transport) receiveDIDComPayload(data []byte) didcomm.Payload {

	pl := mesg.NewPayload(tr.PLPipe.Decrypt(data))
	rm := tr.decryptMsg(pl)

	if pl.Type == pltype.ConnectionError {
		glog.Warning(dto.ToJSON(rm))
	}
	pl.Message = rm
	return &mesg.PayloadImpl{Payload: pl}
}

func (tr Transport) receivePayload(data []byte, originalNonce uint64) *mesg.Payload {
	pl := mesg.NewPayload(tr.PLPipe.Decrypt(data))
	rm := tr.decryptMsg(pl)

	if pl.Type == pltype.ConnectionError {
		glog.Warning(dto.ToJSON(rm))
	}
	// When connection request is started by other end they reset the nonce
	if pl.Type != pltype.ConnectionAck && pl.Type != pltype.ConnectionRequest {
		n := utils.NonceNum(rm.Nonce)
		if originalNonce != n {
			glog.Error("CA comm err, nonce mismatch")
		}
	} else {
		glog.V(5).Infof("nonce values: %v old, %v new\n", originalNonce, rm.Nonce)
	}
	pl.Message = rm
	return pl
}

func (tr Transport) decryptMsg(pl *mesg.Payload) (rm mesg.Msg) {
	if pl.Type == pltype.ConnectionAck ||
		pl.Type == pltype.ConnectionRequest ||
		pl.Type == pltype.ConnectionError ||
		pl.Type == "" {
		return pl.Message
	}
	return *tr.DecMsg(&pl.Message)
}

func (tr Transport) Notify(ws *websocket.Conn, pl *mesg.Payload) error {
	data := tr.PLPipe.Encrypt(dto.ToJSONBytes(pl))
	return websocket.Message.Send(ws, data)
}

func (tr Transport) EncMsg(msg *mesg.Msg) *mesg.Msg {
	return msg.Encrypt(tr.MsgPipe)
}

func (tr Transport) DecMsg(msg *mesg.Msg) *mesg.Msg {
	return msg.Decrypt(tr.MsgPipe)
}

func (tr Transport) EncDIDComMsg(msg didcomm.Msg) didcomm.Msg {
	return msg.Encr(tr.MsgPipe)
}

func (tr Transport) DecDIDComMsg(msg didcomm.Msg) didcomm.Msg {
	return msg.Decr(tr.MsgPipe)
}
