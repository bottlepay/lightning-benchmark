package main

import (
	"context"
	"errors"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"google.golang.org/grpc"
)

type lndConnection struct {
	conn            *grpc.ClientConn
	routerClient    routerrpc.RouterClient
	lightningClient lnrpc.LightningClient
}

func getLndConnection(cfg *lndConfig) (*lndConnection, error) {
	connLogger := log.With("host", cfg.RpcHost)
	senderConn, err := getClientConn(cfg)
	if err != nil {
		return nil, err
	}

	senderClient := lnrpc.NewLightningClient(senderConn)
	var attempt int

	for {
		attempt++
		logger := connLogger.With("attempt", attempt)

		logger.Infow("Attempting to connect to lnd")

		resp, err := senderClient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
		if err == nil {
			if !resp.SyncedToChain {
				logger.Infow("Not synced to chain yet")
				time.Sleep(time.Second)

				continue
			}

			connLogger.Infow("Connected to lnd", "key", resp.IdentityPubkey)
			break
		}

		logger.Infow("Lnd connection attempt failed", "err", err)
		time.Sleep(time.Second)
	}

	return &lndConnection{
		conn:            senderConn,
		routerClient:    routerrpc.NewRouterClient(senderConn),
		lightningClient: lnrpc.NewLightningClient(senderConn),
	}, nil
}

func (l *lndConnection) Close() {
	l.conn.Close()
}

type info struct {
	key    string
	synced bool
}

func (l *lndConnection) GetInfo() (*info, error) {
	infoResp, err := l.lightningClient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	return &info{
		key:    infoResp.IdentityPubkey,
		synced: infoResp.SyncedToChain,
	}, nil
}

func (l *lndConnection) Connect(key, host string) error {
	_, err := l.lightningClient.ConnectPeer(context.Background(), &lnrpc.ConnectPeerRequest{
		Addr: &lnrpc.LightningAddress{
			Host:   host,
			Pubkey: key,
		},
	})
	return err
}

func (l *lndConnection) NewAddress() (string, error) {
	addrResp, err := l.lightningClient.NewAddress(context.Background(), &lnrpc.NewAddressRequest{
		Type: lnrpc.AddressType_WITNESS_PUBKEY_HASH,
	})
	if err != nil {
		return "", err
	}

	return addrResp.Address, nil
}

func (l *lndConnection) OpenChannel(peerKey string, amtSat int64) error {
	_, err := l.lightningClient.OpenChannelSync(context.Background(), &lnrpc.OpenChannelRequest{
		LocalFundingAmount: amtSat,
		NodePubkeyString:   peerKey,
	})
	return err
}

func (l *lndConnection) HasActiveChannels() (bool, error) {
	resp, err := l.lightningClient.ListChannels(context.Background(), &lnrpc.ListChannelsRequest{
		ActiveOnly: true,
	})
	if err != nil {
		return false, err
	}
	return len(resp.Channels) > 0, nil
}

func (l *lndConnection) AddInvoice(amtMsat int64) (string, error) {
	addResp, err := l.lightningClient.AddInvoice(context.Background(), &lnrpc.Invoice{
		ValueMsat: amtMsat,
	})
	if err != nil {
		return "", err
	}
	return addResp.PaymentRequest, nil
}

func (l *lndConnection) SendPayment(invoice string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := l.routerClient.SendPayment(ctx, &routerrpc.SendPaymentRequest{
		PaymentRequest:    invoice,
		TimeoutSeconds:    60,
		NoInflightUpdates: true,
	})
	if err != nil {
		return err
	}

	update, err := stream.Recv()
	if err != nil {
		return err
	}

	if update.State != routerrpc.PaymentState_SUCCEEDED {
		return errors.New("payment failed")
	}

	return nil
}
