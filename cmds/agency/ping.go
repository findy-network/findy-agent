package agency

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"github.com/findy-network/findy-agent/agent/comm"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/utils"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-common-go/agency/client"
	pb "github.com/findy-network/findy-common-go/grpc/ops/v1"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type PingCmd struct {
	cmds.GrpcCmd
	BaseAddr string
}

func (c PingCmd) Validate() error {
	if c.BaseAddr == "" {
		return errors.New("server url cannot be empty")
	}
	return nil
}

func (c PingCmd) RPCExec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	if err = c.GrpcCmd.Validate(); err != nil {
		return nil, err
	}

	baseCfg := client.BuildClientConnBase(c.TLSPath, c.Addr, c.Port, nil)
	conn := client.TryOpen(c.AdminID, baseCfg)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opsClient := pb.NewDevOpsServiceClient(conn)
	result := try.To1(opsClient.Enter(ctx, &pb.Cmd{
		Type: pb.Cmd_PING,
	}))
	cmds.Fprintln(w, "result:", result.GetPing())

	return nil, nil
}

func (c PingCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	p := bytes.NewReader([]byte(""))

	endpointAdd := &endp.Addr{
		BasePath: c.BaseAddr,
		Service:  "/", // use the root as a ping address
	}

	resp := try.To1(comm.SendAndWaitReq(endpointAdd.Address(), p, utils.HTTPReqTimeout))
	cmds.Fprintln(w, string(resp))

	return nil, nil
}
