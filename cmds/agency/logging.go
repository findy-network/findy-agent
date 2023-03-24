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

type LoggingCmd struct {
	cmds.GrpcCmd
	Level string
}

func (c LoggingCmd) Validate() error {
	return c.GrpcCmd.Validate()
}

func (c LoggingCmd) RPCExec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Handle(&err)

	baseCfg := client.BuildClientConnBase(c.TLSPath, c.Addr, c.Port, nil)
	conn := client.TryOpen(c.AdminID, baseCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opsClient := pb.NewDevOpsServiceClient(conn)
	try.To1(opsClient.Enter(ctx, &pb.Cmd{
		Type:    pb.Cmd_LOGGING,
		Request: &pb.Cmd_Logging{Logging: c.Level},
	}))
	cmds.Fprintln(w, "no error")

	return nil, nil
}
