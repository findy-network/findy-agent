package server

import (
	"context"
	"errors"
	"os"

	"github.com/findy-network/findy-agent-api/grpc/ops"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/cmds/onboard"
	"github.com/findy-network/findy-grpc/jwt"
	"github.com/findy-network/findy-wrapper-go/dto"
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

func (a agencyService) PSMHook(hook *ops.DataHook, server ops.Agency_PSMHookServer) error {
	panic("implement me")
}
