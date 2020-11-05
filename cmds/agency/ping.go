package agency

import (
	"context"
	"errors"
	"io"
	"time"

	pb "github.com/findy-network/findy-agent-api/grpc/ops"
	"github.com/findy-network/findy-agent/agent/agency"
	"github.com/findy-network/findy-agent/agent/e2"
	"github.com/findy-network/findy-agent/agent/endp"
	"github.com/findy-network/findy-agent/agent/mesg"
	"github.com/findy-network/findy-agent/cmds"
	"github.com/findy-network/findy-agent/grpc/client"
	"github.com/lainio/err2"
)

type PingCmd struct {
	BaseAddr string
}

func (c PingCmd) Validate() error {
	if c.BaseAddr == "" {
		return errors.New("server url cannot be empty")
	}
	return nil
}

func (c PingCmd) RpcExec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	conn, err := client.OpenClientConn("findy-root", c.BaseAddr)
	err2.Check(err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	opsClient := pb.NewDevOpsClient(conn)
	result, err := opsClient.Enter(ctx, &pb.Cmd{
		Type: pb.Cmd_PING,
	})
	err2.Check(err)
	cmds.Fprintln(w, "result:", result.GetPing())

	return nil, nil
}

func (c PingCmd) Exec(w io.Writer) (r cmds.Result, err error) {
	defer err2.Return(&err)

	p := mesg.Payload{}

	endpointAdd := &endp.Addr{
		BasePath: c.BaseAddr,
		Service:  agency.APIPath,
		PlRcvr:   "ping",
	}

	pl := e2.Payload.Try(cmds.SendAndWaitPayload(&p, endpointAdd, 0))
	cmds.Fprintln(w, "ping ok.",
		"\nserver's host address:", pl.Message.Encrypted,
		"\nversion info:", pl.Message.Name)

	return nil, nil
}
