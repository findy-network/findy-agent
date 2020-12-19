package server

import (
	"context"
	"errors"
	"os"
	"time"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent-api/grpc/ops"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/onboard"
	"github.com/findy-network/findy-grpc/jwt"
	"github.com/findy-network/findy-wrapper-go/dto"
	"github.com/golang/glog"
	"github.com/lainio/err2"
)

type agencyService struct {
	Root string
	ops.UnimplementedAgencyServer
}

func (a agencyService) Onboard(ctx context.Context, onboarding *ops.Onboarding) (st *ops.OnboardResult, err error) {
	defer err2.Return(&err)
	st = &ops.OnboardResult{Ok: false}

	user := jwt.User(ctx)
	if user != a.Root {
		return st, errors.New("access right")
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
		Result: &ops.OnboardResult_OkResult{
			JWT:            jwt.BuildJWT(r.CADID),
			CADID:          r.CADID,
			InvitationJson: dto.ToJSON(r.Invitation),
		},
	}, nil
}

func (a agencyService) PSMHook(hook *ops.DataHook, server ops.Agency_PSMHookServer) (err error) {
	defer err2.Catch(func(err error) {
		glog.Errorf("grpc agent listen error: %s", err)
		status := &ops.AgencyStatus{
			Id: err.Error(),
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

	glog.V(1).Infoln("*-agent PSM listener:", hook.Id)

	go startPermanentPSMCleanup(ctx)
	time.Sleep(100 * time.Millisecond)

	listenKey := bus.AgentKeyType{
		AgentDID: bus.AllAgents,
		ClientID: hook.Id,
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
		DID:   notify.AgentKeyType.AgentDID,
		Nonce: notify.ProtocolID,
	}
	p := e2.PSM.Try(psm.GetPSM(psmKey))
	err2.Check(psm.RmPSM(p))
}

func handleNotify(hook *ops.DataHook, server ops.Agency_PSMHookServer, notify bus.AgentNotify) {
	defer err2.Catch(func(err error) {
		glog.Error(err)
	})

	glog.V(1).Infoln("notification", notify.ID, "arrived")
	pid := &pb.ProtocolID{
		TypeId: protocolType[notify.ProtocolFamily],
		Role:   roleType[notify.Initiator],
		Id:     notify.ProtocolID,
	}

	psmKey := psm.StateKey{
		DID:   notify.AgentKeyType.AgentDID,
		Nonce: notify.ProtocolID,
	}
	agentStatus := ops.AgencyStatus{
		DID:            psmKey.DID,
		Id:             hook.Id,
		ProtocolStatus: tryProtocolStatus(pid, psmKey),
	}
	if hook.Id != notify.ClientID {
		glog.Warningf("client id mismatch: c/s: %s/%s",
			hook.Id, notify.ClientID)
	}
	err2.Check(server.Send(&agentStatus))

	// Update PSM state to trigger immediate cleanup
	err2.Check(prot.AddAndSetFlagUpdatePSM(psmKey,
		psm.Archived,  // set this
		psm.Archiving, // clear this
	))
}
