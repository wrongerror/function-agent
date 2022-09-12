package client

import (
	"context"
	"fmt"
	"github.com/dapr/dapr/pkg/channel"
	dgrpc "github.com/dapr/dapr/pkg/grpc"
	"github.com/pkg/errors"
)

func CreateGRPCChannel(m *dgrpc.Manager, host string, port int, sslEnabled bool) (channel.AppChannel, error) {
	address := fmt.Sprintf("%s:%v", host, port)
	conn, _, err := m.GetGRPCConnection(context.TODO(), address, "", "", true, false, sslEnabled)
	if err != nil {
		return nil, errors.Errorf("error establishing connection to app grpc on address %s: %s", address, err)
	}
	m.AppClient = conn
	return nil, nil
}
