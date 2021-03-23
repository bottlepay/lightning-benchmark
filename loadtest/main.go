package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/lightningnetwork/lnd/build"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon "gopkg.in/macaroon.v2"
)

func getClientConn(ctx *lndConfig) (*grpc.ClientConn, error) {
	creds, err := credentials.NewClientTLSFromFile(ctx.TlsCertPath, "")
	if err != nil {
		return nil, err
	}

	macBytes, err := ioutil.ReadFile(ctx.MacaroonPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read macaroon path (check "+
			"the network setting!): %v", err)
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		return nil, fmt.Errorf("unable to decode macaroon: %v", err)
	}

	cred := macaroons.NewMacaroonCredential(mac)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(cred),
	}

	conn, err := grpc.Dial(ctx.RpcHost, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to RPC server: %v", err)
	}

	return conn, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "load test"
	app.Version = build.Version() + " commit=" + build.Commit
	app.Commands = []cli.Command{loadCommand, setupCommand}
	if err := app.Run(os.Args); err != nil {
		log.Errorw("Exiting", "err", err)
	}
}
