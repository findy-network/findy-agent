package comm

import (
	"fmt"

	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
)

var noEncryptedTypes = map[string]bool{
	pltype.AriesConnectionOffer:           true,
	pltype.SAIssueCredentialAcceptPropose: true,
}

func noCrypto(t string) bool {
	return noEncryptedTypes[t]
}

// ProcessMsg is a helper function to input a decrypted Msg and write output. It
// builds the Payload ready.
func ProcessMsg(packet Packet, handler func(im, om didcomm.Msg) (err error)) (response didcomm.Payload) {
	input := func(payload didcomm.Payload) (im didcomm.Msg, om didcomm.Msg) {
		var decryptedMsg didcomm.Msg
		if noCrypto(payload.Type()) {
			decryptedMsg = payload.Message()
		} else {
			decryptedMsg = packet.Receiver.Trans().DecDIDComMsg(payload.Message())
		}
		om = mesg.MsgCreator.Create(didcomm.MsgInit{Nonce: decryptedMsg.Nonce()}).(didcomm.Msg)
		return decryptedMsg, om
	}
	output := func(om didcomm.Msg) didcomm.Payload {
		var m didcomm.Msg
		if noCrypto(packet.Payload.Type()) {
			m = om
		} else {
			m = packet.Receiver.Trans().EncDIDComMsg(om)
		}
		response = mesg.PayloadCreator.NewMsg(packet.Payload.ID(),
			packet.Payload.Type(), m)
		return response
	}

	im, om := input(packet.Payload)
	err := handler(im, om)
	if err != nil {
		om.SetError(fmt.Sprintf("handling request: %s", err))
	}
	return output(om)
}
