package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/record"
	"google.golang.org/grpc"
)

type lndConnection struct {
	conn            *grpc.ClientConn
	routerClient    routerrpc.RouterClient
	lightningClient lnrpc.LightningClient

	key string
}

func tryFunc(f func() error, maxAttempts int) error {
	var attempts int
	for {
		err := f()
		if err == nil {
			return err
		}

		attempts++
		if attempts == maxAttempts {
			return err
		}

		time.Sleep(time.Second)
	}
}

func getLndConnection(cfg *lndConfig) (*lndConnection, error) {
	logger := log.With("host", cfg.RpcHost)

	var conn *grpc.ClientConn
	err := tryFunc(
		func() error {
			var err error
			conn, err = getClientConn(cfg)
			return err
		}, 10)
	if err != nil {
		return nil, err
	}

	senderClient := lnrpc.NewLightningClient(conn)

	logger.Infow("Attempting to connect to lnd")
	var key string
	for {
		resp, err := senderClient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
		if err == nil {
			if !resp.SyncedToChain {
				time.Sleep(time.Second)

				continue
			}

			key = resp.IdentityPubkey

			logger.Infow("Connected to lnd", "key", resp.IdentityPubkey)
			break
		}

		time.Sleep(time.Second)
	}

	return &lndConnection{
		conn:            conn,
		routerClient:    routerrpc.NewRouterClient(conn),
		lightningClient: lnrpc.NewLightningClient(conn),
		key:             key,
	}, nil
}

func (l *lndConnection) Close() {
	l.conn.Close()
}

func (l *lndConnection) Key() string {
	return l.key
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
		SpendUnconfirmed:   true,
	})
	return err
}

func (l *lndConnection) ActiveChannels() (int, error) {
	resp, err := l.lightningClient.ListChannels(context.Background(), &lnrpc.ListChannelsRequest{
		ActiveOnly: true,
	})
	if err != nil {
		return 0, err
	}
	return len(resp.Channels), nil
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
		FeeLimitSat:       1000,
	})
	if err != nil {
		return err
	}

	update, err := stream.Recv()
	if err != nil {
		return err
	}

	if update.State != routerrpc.PaymentState_SUCCEEDED {
		return fmt.Errorf("payment failed: %v", update.State)
	}

	return nil
}

func (l *lndConnection) SendKeysend(destination string, amtMsat int64) error {
	dest, err := hex.DecodeString(destination)
	if err != nil {
		return err
	}

	var preimage lntypes.Preimage
	if _, err := rand.Read(preimage[:]); err != nil {
		return err
	}
	hash := preimage.Hash()

	var req = routerrpc.SendPaymentRequest{
		PaymentHash:       hash[:],
		Dest:              dest,
		AmtMsat:           amtMsat,
		TimeoutSeconds:    60,
		NoInflightUpdates: true,
		DestCustomRecords: map[uint64][]byte{
			record.KeySendType: preimage[:],
		},
		FinalCltvDelta: 40,
		DestFeatures:   []lnrpc.FeatureBit{lnrpc.FeatureBit_TLV_ONION_OPT},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := l.routerClient.SendPayment(ctx, &req)
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

func (l *lndConnection) HasFunds() error {
	for {
		resp, err := l.lightningClient.WalletBalance(context.Background(), &lnrpc.WalletBalanceRequest{})
		if err != nil {
			return err
		}
		if resp.ConfirmedBalance > 0 {
			return nil
		}

		time.Sleep(time.Second)
	}
}
