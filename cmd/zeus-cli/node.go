package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/formicidae-tracker/zeus/internal/zeus"
	"github.com/formicidae-tracker/zeus/pkg/zeuspb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Node holds connection information for an available zeus server. It
// also expose an interface for one shot RPC call to zeuspb server.
type Node struct {
	Name    string
	Address string
	Port    int
}

func closeAndLogError(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		logrus.WithError(err).Error("gRPC Close() failure")
	}
}

func (n Node) DialAddress() string {
	return fmt.Sprintf("%s:%d", n.Address, n.Port)
}

func (n Node) Connect() (conn *grpc.ClientConn, client zeuspb.ZeusClient, err error) {
	defer func() {
		if err == nil || conn == nil {
			return
		}
		closeAndLogError(conn)
		conn = nil
	}()
	conn, err = grpc.Dial(
		n.DialAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	return conn, zeuspb.NewZeusClient(conn), nil

}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	st := status.Convert(err)
	return errors.New(st.Message())
}
func (n Node) Status(ctx context.Context) (*zeuspb.Status, error) {
	conn, client, err := n.Connect()
	if err != nil {
		return nil, err
	}
	defer closeAndLogError(conn)
	st, err := client.GetStatus(ctx, &zeuspb.Empty{})
	return st, mapError(err)
}

func (n Node) StartClimate(ctx context.Context, seasonFileContent []byte) error {
	conn, client, err := n.Connect()
	if err != nil {
		return err
	}
	defer closeAndLogError(conn)
	_, err = client.StartClimate(ctx,
		&zeuspb.StartRequest{
			SeasonFile: string(seasonFileContent),
			Version:    zeus.ZEUS_VERSION,
		})
	return mapError(err)
}

func (n Node) StopClimate(ctx context.Context) error {
	conn, client, err := n.Connect()
	if err != nil {
		return err
	}
	defer closeAndLogError(conn)
	_, err = client.StopClimate(ctx, &zeuspb.Empty{})
	return mapError(err)
}
