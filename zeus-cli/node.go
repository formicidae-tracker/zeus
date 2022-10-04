package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/formicidae-tracker/zeus"
	"github.com/formicidae-tracker/zeus/zeuspb"
	"google.golang.org/grpc"
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
		log.Printf("gRPC close() failure: %s", err)
	}
}

func (n Node) Connect() (conn *grpc.ClientConn, client zeuspb.ZeusClient, err error) {
	defer func() {
		if err == nil || conn == nil {
			return
		}
		closeAndLogError(conn)
		conn = nil
	}()
	conn, err = grpc.Dial(fmt.Sprintf("%s:%d", n.Address, n.Port))
	if err != nil {
		return nil, nil, err
	}

	return conn, zeuspb.NewZeusClient(conn), nil

}

func (n Node) Status() (*zeuspb.Status, error) {
	conn, client, err := n.Connect()
	if err != nil {
		return nil, err
	}
	defer closeAndLogError(conn)
	return client.GetStatus(context.Background(), &zeuspb.Empty{})
}

func (n Node) StartClimate(seasonFileContent []byte) error {
	conn, client, err := n.Connect()
	if err != nil {
		return err
	}
	defer closeAndLogError(conn)
	_, err = client.StartClimate(context.Background(),
		&zeuspb.StartRequest{
			SeasonFile: string(seasonFileContent),
			Version:    zeus.ZEUS_VERSION,
		})
	return err
}

func (n Node) StopClimate() error {
	conn, client, err := n.Connect()
	if err != nil {
		return err
	}
	defer closeAndLogError(conn)
	_, err = client.StopClimate(context.Background(), &zeuspb.Empty{})
	return err
}
