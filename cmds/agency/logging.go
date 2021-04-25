package agency

import (
	"context"
	"io"
	"time"

	pb "github.com/findy-network/findy-agent-api/grpc/ops/v1"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-common-go/agency/client"
	"github.com/lainio/err2"
)

type LoggingCmd struct {
	cmds.GrpcCmd
	Level string
}

func (c LoggingCmd) Validate() error {
	if err := c.GrpcCmd.Validate(); err != nil {
		return err
	}
	return nil
}

func (c LoggingCmd) RpcExec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	baseCfg := client.BuildClientConnBase(c.TlsPath, c.Addr, c.Port, nil)
	conn := client.TryOpen(c.AdminID, baseCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opsClient := pb.NewDevOpsServiceClient(conn)
	err2.Empty.Try(opsClient.Enter(ctx, &pb.Cmd{
		Type:    pb.Cmd_LOGGING,
		Request: &pb.Cmd_Logging{Logging: c.Level},
	}))
	err2.Check(err)
	cmds.Fprintln(w, "no error")

	return nil, nil
}
