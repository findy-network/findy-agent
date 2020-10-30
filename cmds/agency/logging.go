package agency

import (
	"context"
	"errors"
	"io"
	"time"

	pb "github.com/findy-network/findy-agent-api/grpc/agency"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/grpc/client"
	"github.com/lainio/err2"
)

type LoggingCmd struct {
	BaseAddr string
	Level    string
}

func (c LoggingCmd) Validate() error {
	if c.BaseAddr == "" {
		return errors.New("server url cannot be empty")
	}
	return nil
}

func (c LoggingCmd) RpcExec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	conn, err := client.OpenClientConn("findy-root", c.BaseAddr)
	err2.Check(err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opsClient := pb.NewDevOpsClient(conn)
	err2.Empty.Try(opsClient.Enter(ctx, &pb.Cmd{
		Type:    pb.Cmd_LOGGING,
		Request: &pb.Cmd_Logging{Logging: c.Level},
	}))
	err2.Check(err)
	cmds.Fprintln(w, "no error")

	return nil, nil
}
