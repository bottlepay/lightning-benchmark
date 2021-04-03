package main

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/urfave/cli"
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

	log.Infow("Attempting to connect to bitcoind")

	for {
		_, err := bitcoindConn.GetBlockChainInfo()
		if err == nil {
			log.Infow("Connected to bitcoind")
			return bitcoindConn, nil
		}

		time.Sleep(time.Second)
	}
}

func setup(_ *cli.Context) error {
	cfg, err := loadConfig()
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

	log.Infow("Creating bitcoind wallet")
	_, err = bitcoindConn.CreateWallet("")
	if err != nil {
		return err
	}

	for _, nodes := range cfg.Nodes {
		log.Infow("Fund sender")
		senderClient, err := getNodeConnection(&nodes.Sender)
		if err != nil {
			return err
		}
		defer senderClient.Close()

		addrResp, err := senderClient.NewAddress()
		if err != nil {
			return err
		}
		log.Infow("Generated funding address", "address", addrResp)

		senderAddr, err := btcutil.DecodeAddress(addrResp, &chaincfg.RegressionNetParams)
		if err != nil {
			return err
		}
		_, err = bitcoindConn.GenerateToAddress(1, senderAddr, nil)
		if err != nil {
			return err
		}

		log.Infow("Mature coin")
		_, err = bitcoindConn.GenerateToAddress(105, addr, nil)
		if err != nil {
			return err
		}

		log.Infow("Wait for coin to appear in wallet")
		if err := senderClient.HasFunds(); err != nil {
			return err
		}

		receiverClient, err := getNodeConnection(&nodes.Receiver)
		if err != nil {
			return err
		}
		defer receiverClient.Close()

		infoResp, err := receiverClient.GetInfo()
		if err != nil {
			return err
		}
		receiverKey := infoResp.key
		log.Infow("Receiver info", "pubkey", receiverKey)

		log.Infow("Connecting peers")
		err = senderClient.Connect(receiverKey, nodes.Receiver.Host)
		if err != nil {
			return err
		}

		// Open channels. Because the sender will always choose the channel with
		// the highest balance, the channel will be utilized roughly equally.
		log.Infow("Open channels", "channel_count", cfg.Channels, "capacity_sat", cfg.ChannelCapacitySat)
		for i := 0; i < cfg.Channels; i++ {
			err = senderClient.OpenChannel(receiverKey, cfg.ChannelCapacitySat)
			if err != nil {
				return err
			}
		}

		log.Infow("Confirm channels")
		_, err = bitcoindConn.GenerateToAddress(6, addr, nil)
		if err != nil {
			return err
		}

		log.Infow("Waiting for channels to become active")
		for {
			activeChannels, err := senderClient.ActiveChannels()
			if err != nil {
				return err
			}
			if activeChannels == cfg.Channels {
				break
			}
			time.Sleep(time.Second)
		}

		log.Infow("Channels active")
	}
	return nil
}
