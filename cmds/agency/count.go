package agency

import (
	"context"
	"io"
	"time"

	pb "github.com/findy-network/findy-agent-api/grpc/ops"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-common-go/agency/client"
	"github.com/lainio/err2"
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

func (c CountCmd) RpcExec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	baseCfg := client.BuildClientConnBase(c.TlsPath, c.Addr, c.Port, nil)
	conn := client.TryOpen(c.AdminID, baseCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opsClient := pb.NewDevOpsClient(conn)
	result, err := opsClient.Enter(ctx, &pb.Cmd{
		Type: pb.Cmd_COUNT,
	})
	err2.Check(err)
	cmds.Fprintln(w, "count result:\n", result.GetCount())

	return nil, nil
}
