package server

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/bus"
	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/handshake"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/agent/prot"
	"github.com/findy-network/findy-agent/agent/psm"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/enclave"
	"github.com/findy-network/findy-agent/method"
	pb "github.com/findy-network/findy-common-go/grpc/agency/v1"
	ops "github.com/findy-network/findy-common-go/grpc/ops/v1"
	"github.com/findy-network/findy-common-go/jwt"
	"github.com/golang/glog"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

type agencyService struct {
	Root string
	ops.UnimplementedAgencyServiceServer
}

func (a agencyService) Onboard(
	ctx context.Context,
	onboarding *ops.Onboarding,
) (
	st *ops.OnboardResult,
	err error,
) {
	defer err2.Annotate("CA Onboard API", &err)
	st = &ops.OnboardResult{Ok: false}

	user := jwt.User(ctx)
	if user != a.Root {
		return st, errors.New("access right")
	}

	if enclave.WalletKeyExists(onboarding.Email) {
		glog.Warningln("user already registered by:", onboarding.Email)
		return st, errors.New("invalid user")
	}

	agentName := strings.Replace(onboarding.Email, "@", "_", -1)
	ac := try.To1(handshake.AnchorAgent(onboarding.Email, onboarding.PublicDIDSeed))

	caDID := ac.NewDID(method.TypeSov, "")
	DIDStr := caDID.Did()
	caVerKey := caDID.VerKey()

	agency.AddHandler(DIDStr, ac)
	agency.Register.Add(ac.RootDid().Did(), agentName, DIDStr, caVerKey)

	ac.SetMyDID(caDID)

	agency.SaveRegistered()
	glog.V(2).Infoln("build onboarding grpc result:",
		agentName, DIDStr)

	return &ops.OnboardResult{
		Ok: true,
		Result: &ops.OnboardResult_OKResult{
			JWT:            jwt.BuildJWTWithLabel(DIDStr, onboarding.Email),
			CADID:          DIDStr,
			InvitationJSON: "{}",
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
	ctx := try.To1(jwt.CheckTokenValidity(server.Context()))
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
	p := try.To1(psm.GetPSM(psmKey))
	try.To(psm.RmPSM(p))
}

func handleNotify(hook *ops.DataHook, server ops.AgencyService_PSMHookServer, notify bus.AgentNotify) {
	defer err2.Catch(func(err error) {
		glog.Errorln("ERROR in psm hook notify handler", err)
	})

	glog.V(1).Infoln("notification", notify.ID, "arrived")
	pid := &pb.ProtocolID{
		TypeID: pltype.ProtocolTypeForFamily(notify.ProtocolFamily),
		Role:   notify.Role,
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
	try.To(server.Send(&agentStatus))

	// Update PSM state to trigger immediate cleanup
	try.To(prot.AddAndSetFlagUpdatePSM(psmKey,
		psm.Archived,  // set this
		psm.Archiving, // clear this
	))
}

func tryCaDID(psmKey psm.StateKey) string {
	waReceiver := comm.ActiveRcvrs.Get(psmKey.DID)
	myCA := waReceiver.MyCA()
	assert.D.True(myCA != nil, "we must have CA for our WA")
	caDID := myCA.MyDID().Did()
	assert.D.True(caDID != "", "we must get CA DID for API caller")
	return caDID
}
