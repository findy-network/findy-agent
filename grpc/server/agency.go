package server

import (
	"context"
	"errors"
	"os"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent-api/grpc/ops"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/e2"
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
		//status := &pb.AgentStatus{
		//	ClientId: &pb.ClientID{Id: clientID.Id},
		//}
		//if err := server.Send(status); err != nil {
		//	glog.Errorln("error sending response:", err)
		//}
	})
	ctx := e2.Ctx.Try(jwt.CheckTokenValidity(server.Context()))
	user := jwt.User(ctx)
	if user != a.Root {
		return errors.New("access right")
	}

	glog.V(1).Infoln("*-agent PSM listener:", hook.Id)

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
				Id:             hook.Id,
				ProtocolStatus: protocolStatus(pid, psmKey),
				//Notification: &pb.Notification{
				//	Id:             notify.ID,
				//	TypeId:         notificationTypeID[notify.NotificationType],
				//	ConnectionId:   notify.ConnectionID,
				//	ProtocolId:     notify.ProtocolID,
				//	ProtocolFamily: notify.ProtocolFamily,
				//	ProtocolType:   protocolType[notify.ProtocolFamily],
				//	Timestamp:      notify.Timestamp,
				//	Role:           roleType[notify.Initiator],
				//},
			}
			if hook.Id != notify.ClientID {
				glog.Warningf("client id mismatch: c/s: %s/%s",
					hook.Id, notify.ClientID)
			}
			//agentStatus.ClientId.Id = notify.ClientID
			err2.Check(server.Send(&agentStatus))
		case <-ctx.Done():
			glog.V(1).Infoln("ctx.Done() received, returning")
			break loop
		}
	}

	return nil
}
