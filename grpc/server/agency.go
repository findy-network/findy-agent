package server

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/onboard"
	"github.com/findy-network/findy-agent/enclave"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	ops "github.com/findy-network/findy-common-go/grpc/ops/v1"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
)

type agencyService struct {
	Root string
	ops.UnimplementedAgencyServiceServer
}

func (a agencyService) Onboard(ctx context.Context, onboarding *ops.Onboarding) (st *ops.OnboardResult, err error) {
	defer err2.Return(&err)
	st = &ops.OnboardResult{Ok: false}

	user := jwt.User(ctx)
	if user != a.Root {
		return st, errors.New("access right")
	}

	if enclave.WalletKeyExists(onboarding.Email) {
		glog.Warningln("user already registered by:", onboarding.Email)
		return st, errors.New("invalid user")
	}

	r, err := onboard.Cmd{
		Cmd: cmds.Cmd{
			WalletName: utils.Settings.WebOnboardWalletName(),
			WalletKey:  utils.Settings.WebOnboardWalletKey(),
		},
		Email:      onboarding.Email,
		AgencyAddr: utils.Settings.HostAddr(),
	}.Exec(os.Stdout)
	err2.Check(err)

	return &ops.OnboardResult{
		Ok: true,
		Result: &ops.OnboardResult_OKResult{
			JWT:            jwt.BuildJWTWithLabel(r.CADID, onboarding.Email),
			CADID:          r.CADID,
			InvitationJSON: dto.ToJSON(r.Invitation),
		},
	}, nil
}

func (a agencyService) PSMHook(hook *ops.DataHook, server ops.AgencyService_PSMHookServer) (err error) {
	defer err2.Catch(func(err error) {
		glog.Errorf("grpc agent listen error: %s", err)
		status := &ops.AgencyStatus{
			ID: err.Error(),
		}
		if err := server.Send(status); err != nil {
			glog.Errorln("error sending response:", err)
		}
	})
	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	user := jwt.User(ctx)
	if user != a.Root {
		return errors.New("access right")
	}

	glog.V(1).Infoln("*-agent PSM listener:", hook.ID)

	go startPermanentPSMCleanup(ctx)
	time.Sleep(100 * time.Millisecond)

	listenKey := bus.AgentKeyType{
		AgentDID: bus.AllAgents,
		ClientID: hook.ID,
	}
	notifyChan := bus.WantAllAgencyActions.AgentAddListener(listenKey)
	defer bus.WantAllAgencyActions.AgentRmListener(listenKey)

loop:
	for {
		select {
		case notify := <-notifyChan:
			handleNotify(hook, server, notify)

		case <-ctx.Done():
			glog.V(0).Infoln("ctx.Done() received, returning")
			break loop
		}
	}

	return nil
}

//startPermanentPSMCleanup
func startPermanentPSMCleanup(ctx context.Context) {
	clientID := utils.UUID()
	listenKey := bus.AgentKeyType{
		AgentDID: bus.AllAgents,
		ClientID: clientID,
	}
	notifyChan := bus.WantAllPSMCleanup.AgentAddListener(listenKey)
	defer bus.WantAllPSMCleanup.AgentRmListener(listenKey)

loop:
	for {
		select {
		case cleanupNotify := <-notifyChan:
			handleCleanupNotify(cleanupNotify)

		case <-ctx.Done():
			glog.V(0).Infoln(
				"startPermanentPSMCleanup ctx.Done() received, returning")
			break loop
		}
	}

}

func handleCleanupNotify(notify bus.AgentNotify) {
	defer err2.Catch(func(err error) {
		glog.Error(err)
	})

	glog.V(1).Infoln("cleanup notification", notify.ID, "arrived")

	psmKey := psm.StateKey{
		DID:   notify.AgentDID,
		Nonce: notify.ProtocolID,
	}
	p := e2.PSM.Try(psm.GetPSM(psmKey))
	err2.Check(psm.RmPSM(p))
}

func handleNotify(hook *ops.DataHook, server ops.AgencyService_PSMHookServer, notify bus.AgentNotify) {
	defer err2.Catch(func(err error) {
		glog.Errorln("ERROR in psm hook notify handler", err)
	})

	glog.V(1).Infoln("notification", notify.ID, "arrived")
	pid := &pb.ProtocolID{
		TypeID: protocolType[notify.ProtocolFamily],
		Role:   roleType[notify.Initiator],
		ID:     notify.ProtocolID,
	}

	psmKey := psm.StateKey{
		DID:   notify.AgentDID,
		Nonce: notify.ProtocolID,
	}

	status, connID := tryProtocolStatus(pid, psmKey)
	glog.Infoln("connID:", connID)
	agentStatus := ops.AgencyStatus{
		DID:            tryCaDID(psmKey),
		ID:             hook.ID,
		ProtocolStatus: status,
		ConnectionID:   connID,
	}
	if hook.ID != notify.ClientID {
		glog.Warningf("client id mismatch: c/s: %s/%s",
			hook.ID, notify.ClientID)
	}
	err2.Check(server.Send(&agentStatus))

	// Update PSM state to trigger immediate cleanup
	err2.Check(prot.AddAndSetFlagUpdatePSM(psmKey,
		psm.Archived,  // set this
		psm.Archiving, // clear this
	))
}

func tryCaDID(psmKey psm.StateKey) string {
	waReceiver := comm.ActiveRcvrs.Get(psmKey.DID)
	myCA := waReceiver.MyCA()
	assert.D.True(myCA != nil, "we must have CA for our WA")
	caDID := myCA.Trans().PayloadPipe().In.Did()
	assert.D.True(caDID != "", "we must get CA DID for API caller")
	return caDID
}
