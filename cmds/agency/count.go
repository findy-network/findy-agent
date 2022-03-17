package agency

import (
	"context"
	"io"
	"time"

	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-common-go/agency/client"
	pb "github.com/findy-network/findy-common-go/grpc/ops/v1"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type CountCmd struct {
	cmds.GrpcCmd
	Level string
}

func (c CountCmd) Validate() (err error) {
	if err := c.GrpcCmd.Validate(); err != nil {
		return err
	}
	return nil
}

func (c CountCmd) RPCExec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	baseCfg := client.BuildClientConnBase(c.TLSPath, c.Addr, c.Port, nil)
	conn := client.TryOpen(c.AdminID, baseCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opsClient := pb.NewDevOpsServiceClient(conn)
	result := try.To1(opsClient.Enter(ctx, &pb.Cmd{
		Type: pb.Cmd_COUNT,
	}))
	cmds.Fprintln(w, "count result:\n", result.GetCount())

	return nil, nil
}
