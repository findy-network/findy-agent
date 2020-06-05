package caapi

import (
	"fmt"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/service"
	"github.com/findy-network/findy-agent/agent/ssi"
	"github.com/findy-network/findy-agent/agent/status"
	"github.com/findy-network/findy-agent/agent/utils"
	didexchange "github.com/findy-network/findy-agent/std/didexchange/invitation"
	"github.com/findy-network/findy-wrapper-go"
	"github.com/findy-network/findy-wrapper-go/anoncreds"
	"github.com/findy-network/findy-wrapper-go/did"
	"github.com/findy-network/findy-wrapper-go/ledger"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

var caHandlers = map[string]comm.PlHandlerFunc{
	// Task API commands
	pltype.CATaskReady:  doCATaskReady,
	pltype.CATaskStatus: doCATask,
	pltype.CATaskList:   doCATask,

	// Making DIDComm+ other protocol starting commands
	pltype.CAPairwiseInvitation:   doCAInitConnection,
	pltype.CAPairwiseCreate:       doCAStartProtocolWithInvitation,
	pltype.CAPairwiseAndCredReq:   doCAStartPairwiseAndCredReq,
	pltype.CAPairwiseAndProofProp: doCAStartPairwiseAndProofPropose,
	pltype.CAPairwiseAndTrustPing: doCAStartPairwiseAndTrustPing,

	// During DIDComm started protocol starting commands
	pltype.CATrustPing:    doCAStartProtocol,
	pltype.CACredRequest:  doCAStartProtocol,
	pltype.CACredOffer:    doCAStartProtocol,
	pltype.CAProofPropose: doCAStartProtocol,
	pltype.CAProofRequest: doCAStartProtocol,
	pltype.CABasicMessage: doCAStartProtocol,

	// Continue messages for aries protocols
	pltype.CAContinuePresentProofProtocol:    doCAContinueProtocol,
	pltype.CAContinueIssueCredentialProtocol: doCAContinueProtocol,

	// Generic helpers
	pltype.CAWalletGet:        doCAWalletGet,
	pltype.CALedgerWriteDid:   doCALedgerWriteDid,
	pltype.CAPingOwnCA:        doCAPingOwnCA,
	pltype.CAPingAPIEndp:      doPingAPIEndp,
	pltype.CAAttachAPIEndp:    doAttachAPIEndp,
	pltype.CAAttachEADefImpl:  doAttachSAImpl,
	pltype.CALedgerGetSchema:  doCALedgerGetSchema,
	pltype.CALedgerGetCredDef: doCALedgerGetCredDef,
	pltype.CADIDVerKey:        doCADIDVerKey,
	pltype.CACredDefCreate:    doCACredDefCreate,
	pltype.CASchemaCreate:     doCASchemaCreate,
	pltype.CAGetJWT:           doCAGetJWT,
}

func init() {
	comm.CloudAgentAPI().Add(caHandlers)
}

func doCAStartPairwiseAndCredReq(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		startPwPlus(packet, im, om, pltype.CACredRequest)
		return nil
	})
}

func doCAStartPairwiseAndProofPropose(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		startPwPlus(packet, im, om, pltype.CAProofPropose)
		return nil
	})
}

func doCAStartPairwiseAndTrustPing(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		startPwPlus(packet, im, om, pltype.CATrustPing)
		return nil
	})
}

func startPwPlus(packet comm.Packet, im, om didcomm.Msg, next string) {
	// Set PL type to CAPairwiseCreate to allow next function to
	// find correct protocol
	packet.Payload.SetType(pltype.CAPairwiseCreate)
	tID := prot.FindAndStart(packet, im, om, im.ConnectionInvitation().ID)

	// register monitor to tell us when first protocol will be ready
	key := psm.NewStateKey(packet.Receiver.WorkerEA(), tID)
	done := bus.ReadyStation.StartListen(key)

	// This Task ID goes to client to wait for, not the tID.
	theTaskID := utils.UUID() // this is UUID version
	om.SetSubLevelID(theTaskID)

	// kick off the listener to start second protocol, we'll wait it here
	go func() {
		defer err2.CatchTrace(func(err error) {}) // dont let crash on panics

		ok := <-done // previously registered monitoring channel

		if ok {
			glog.V(2).Info("----- chained protocol start -----")
			packet.Payload.SetType(next)
			prot.FindAndStart(packet, im, om, theTaskID)
		} else {
			glog.V(2).Info("----- chained protocol CANNOT start -----")
			meDID := packet.Receiver.Trans().MessagePipe().In.Did()
			t := comm.Task{
				TypeID: packet.Payload.Type(),
				Nonce:  theTaskID,
			}
			wpl := mesg.NewPayloadBase(t.Nonce, t.TypeID)
			err2.Check(prot.UpdatePSM(meDID, "", &t, wpl, psm.ReadyNACK))
		}
	}()
}

func doCAInitConnection(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		t := prot.InitTask(packet, im, om)
		endpoint := packet.Receiver.CAEndp(true)

		om.SetInvitation(&didexchange.Invitation{
			ID:              t.Nonce,
			Type:            pltype.AriesConnectionInvitation,
			Label:           t.Info,
			ServiceEndpoint: endpoint.Address(),
			RecipientKeys:   []string{endpoint.VerKey},
		})
		return nil
	})
}

func doCAStartProtocolWithInvitation(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		prot.FindAndStart(packet, im, om, im.ConnectionInvitation().ID)
		return nil
	})
}

func doCAStartProtocol(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		prot.FindAndStart(packet, im, om, "")
		return nil
	})
}

func doCAContinueProtocol(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		prot.Continue(packet, im)
		return nil
	})
}

func doCATaskReady(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		om.SetNonce(im.Nonce()) // Let's keep Nonce
		w := status.NewWorker(packet.Receiver)
		ready, err := w.TaskReady(&status.TaskParam{
			Type: packet.Payload.Type(),
			ID:   im.SubLevelID(),
		})
		om.SetReady(ready)
		if err != nil {
			om.SetError(err.Error())
		}
		return nil
	})
}

func doCATask(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		om.SetNonce(im.Nonce()) // Let's keep Nonce
		w := status.NewWorker(packet.Receiver)
		om.SetBody(
			w.Exec(&status.TaskParam{
				Type:        packet.Payload.Type(),
				ID:          im.SubLevelID(),
				DeviceToken: im.Info(),        // FYI currently used for task list
				TsSinceMs:   im.TimestampMs(), // FYI currently used for task list
			}))
		return nil
	})
}

func doCAWalletGet(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		om.SetEndpoint(service.Addr{Endp: packet.Receiver.WorkerEA().ExportWallet(im.VerKey(), "")})
		return nil
	})
}

func doCALedgerWriteDid(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		defer err2.Annotate("write DID", &err)

		targetDID := ssi.NewDid(im.Did(), im.VerKey())
		err2.Check(packet.Receiver.SendNYM(targetDID,
			packet.Receiver.RootDid().Did(),
			findy.NullString, findy.NullString))
		return nil
	})
}

func doCAPingOwnCA(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		a := packet.Receiver
		om.SetEndpoint(a.CAEndp(true).AE())
		return nil
	})
}

func doAttachAPIEndp(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		return packet.Receiver.AttachAPIEndp(im.ReceiverEP())
	})
}

func doPingAPIEndp(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		defer err2.Return(&err)

		glog.V(1).Info("calling sa ping")
		ask, err := packet.Receiver.CallEA(pltype.SAPing, im)
		err2.Check(err)

		om.SetReady(ask.Ready())
		om.SetInfo(ask.Info())

		return err
	})
}

func doAttachSAImpl(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		a := packet.Receiver
		a.AttachSAImpl(im.SubLevelID())
		return nil
	})
}

func doCALedgerGetSchema(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		defer err2.Annotate("process get cred def", &err)
		a := packet.Receiver
		schemaID := im.SubLevelID()
		sID, schema, err := ledger.ReadSchema(a.Pool(), a.RootDid().Did(), schemaID)
		err2.Check(err)
		om.SetSubLevelID(sID)
		om.SetSubMsg(mesg.SubFromJSON(schema))
		return nil
	})
}

func doCALedgerGetCredDef(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		a := packet.Receiver
		credDefID := im.SubLevelID()
		def, err := ssi.CredDefFromLedger(a.RootDid().Did(), credDefID)
		if err != nil {
			return err
		}
		om.SetSubMsg(mesg.SubFromJSON(def))
		return nil
	})
}

func doCADIDVerKey(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		r := <-did.Key(packet.Receiver.Pool(), packet.Receiver.Wallet(), im.Did())
		if r.Err() != nil {
			return fmt.Errorf("getting key in cloud agent: %s", r.Error())
		}
		om.SetDid(im.Did())
		om.SetVerKey(r.Str1())
		return nil
	})
}

func doCACredDefCreate(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		defer err2.Annotate("process cred def create", &err)

		ca := packet.Receiver
		err2.Check(im.Schema().FromLedger(ca.RootDid().Did()))
		tag := im.Info()
		r := <-anoncreds.IssuerCreateAndStoreCredentialDef(
			ca.Wallet(), ca.RootDid().Did(), im.Schema().Stored.Str2(),
			tag, findy.NullString, findy.NullString)
		err2.Check(r.Err())
		om.SetSubLevelID(r.Str1())
		cd := r.Str2()
		err = ledger.WriteCredDef(ca.Pool(), ca.Wallet(), ca.RootDid().Did(), cd)
		return err
	})
}

func doCASchemaCreate(packet comm.Packet) didcomm.Payload {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		defer err2.Annotate("create schema", &err)

		ca := packet.Receiver
		err2.Check(im.Schema().Create(ca.RootDid().Did()))
		err2.Check(im.Schema().ToLedger(ca.Wallet(), ca.RootDid().Did()))
		om.SetSchema(im.Schema())
		om.Schema().ID = im.Schema().ValidID()
		return nil
	})
}

// doCAGetJWT creates and returns a JWT token for gRPC connections.
func doCAGetJWT(packet comm.Packet) (opl didcomm.Payload) {
	return comm.ProcessMsg(packet, func(im, om didcomm.Msg) (err error) {
		//om.SetSubLevelID(jwt.BuildJWT(packet.Receiver.Trans().MessagePipe().In.Did()))
		return nil
	})
}
