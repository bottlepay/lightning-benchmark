package main

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

var setupCommand = cli.Command{
	Name:   "setup",
	Action: setup,
}

func getBitcoindConnection(cfg *bitcoindConfig) (*rpcclient.Client, error) {
	connConfig := rpcclient.ConnConfig{
		Host:                 cfg.Host,
		User:                 cfg.User,
		Pass:                 cfg.Pass,
		DisableConnectOnNew:  true,
		DisableAutoReconnect: false,
		DisableTLS:           true,
		HTTPPostMode:         true,
	}

	bitcoindConn, err := rpcclient.New(&connConfig, nil)
	if err != nil {
		log.Errorw("New rpc connection", "err", err)
		return nil, err
	}

	var attempt int
	for {
		attempt++
		log.Infow("Attempting to connect to bitcoind", "attempt", attempt)

		_, err := bitcoindConn.GetBlockChainInfo()
		if err == nil {
			log.Infow("Connected to bitcoind")
			return bitcoindConn, nil
		}

		log.Infow("Bitcoind connection attempt failed", "err", err)
		time.Sleep(time.Second)
	}
}

func getLndConnection(cfg *lndConfig) (*grpc.ClientConn, error) {
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
				log.Infow("Not synced to chain yet")
				time.Sleep(time.Second)

				continue
			}

			logger.Infow("Connected to lnd", "key", resp.IdentityPubkey)
			return senderConn, nil
		}

		logger.Infow("Lnd connection attempt failed", "err", err)
		time.Sleep(time.Second)
	}
}

func setup(_ *cli.Context) error {
	yamlFile, err := ioutil.ReadFile("loadtest.yml")
	if err != nil {
		return err
	}

	var cfg config
	err = yaml.UnmarshalStrict(yamlFile, &cfg)
	if err != nil {
		return err
	}

	bitcoindConn, err := getBitcoindConnection(&cfg.Bitcoind)
	if err != nil {
		return err
	}

	addr, err := btcutil.DecodeAddress("bcrt1qlppjvkglr9hrznfnx94n4np53axcekzer9dkmv", &chaincfg.RegressionNetParams)
	if err != nil {
		return err
	}
	log.Infow("Using dummy address", "address", addr.String())

	log.Infow("Activate segwit")
	_, err = bitcoindConn.GenerateToAddress(400, addr, nil)
	if err != nil {
		return err
	}

	log.Infow("Fund sender")
	senderConn, err := getLndConnection(&cfg.Sender.Lnd)
	if err != nil {
		return err
	}
	defer senderConn.Close()
	senderClient := lnrpc.NewLightningClient(senderConn)
	addrResp, err := senderClient.NewAddress(context.Background(), &lnrpc.NewAddressRequest{
		Type: lnrpc.AddressType_WITNESS_PUBKEY_HASH,
	})
	if err != nil {
		return err
	}
	senderAddr, err := btcutil.DecodeAddress(addrResp.Address, &chaincfg.RegressionNetParams)
	if err != nil {
		return err
	}
	_, err = bitcoindConn.GenerateToAddress(1, senderAddr, nil)
	if err != nil {
		return err
	}

	log.Infow("Mature coin")
	_, err = bitcoindConn.GenerateToAddress(100, addr, nil)
	if err != nil {
		return err
	}

	receiverConn, err := getLndConnection(&cfg.Receiver.Lnd)
	if err != nil {
		return err
	}
	defer receiverConn.Close()
	receiverClient := lnrpc.NewLightningClient(receiverConn)

	infoResp, err := receiverClient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return err
	}
	receiverKey := infoResp.IdentityPubkey
	log.Infow("Receiver info", "pubkey", receiverKey)

	log.Infow("Connecting peers")
	_, err = senderClient.ConnectPeer(context.Background(), &lnrpc.ConnectPeerRequest{
		Addr: &lnrpc.LightningAddress{
			Host:   cfg.Receiver.Lnd.Host,
			Pubkey: receiverKey,
		},
	})
	if err != nil {
		return err
	}

	log.Infow("Open channel")
	_, err = senderClient.OpenChannelSync(context.Background(), &lnrpc.OpenChannelRequest{
		LocalFundingAmount: 10000000,
		NodePubkeyString:   receiverKey,
	})
	if err != nil {
		return err
	}

	log.Infow("Confirm channel")
	_, err = bitcoindConn.GenerateToAddress(6, addr, nil)
	if err != nil {
		return err
	}

	log.Infow("Waiting for channel to become active")
	for {
		resp, err := senderClient.ListChannels(context.Background(), &lnrpc.ListChannelsRequest{
			ActiveOnly: true,
		})
		if err == nil && len(resp.Channels) > 0 {
			break
		}
		time.Sleep(time.Second)
	}

	log.Infow("Channel active")
	return nil
}
