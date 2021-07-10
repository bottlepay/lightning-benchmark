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

	type node struct {
		client nodeInterface
		host   string
	}

	var nodes []*node

	senderClient, err := getNodeConnection(&cfg.Sender)
	if err != nil {
		return err
	}
	defer senderClient.Close()
	nodes = append(nodes,
		&node{
			client: senderClient,
			host:   cfg.Sender.Host,
		},
	)

	if cfg.Router.Host != "" {
		routerClient, err := getNodeConnection(&cfg.Router)
		if err != nil {
			return err
		}
		defer routerClient.Close()
		nodes = append(nodes,
			&node{
				client: routerClient,
				host:   cfg.Router.Host,
			},
		)
	}

	receiverClient, err := getNodeConnection(&cfg.Receiver)
	if err != nil {
		return err
	}
	defer receiverClient.Close()
	nodes = append(nodes,
		&node{
			client: receiverClient,
			host:   cfg.Receiver.Host,
		},
	)

	log.Infow("Fund wallets")
	for _, n := range nodes[:len(nodes)-1] {
		addrResp, err := n.client.NewAddress()
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
	}

	log.Infow("Mature coin")
	_, err = bitcoindConn.GenerateToAddress(105, addr, nil)
	if err != nil {
		return err
	}

	log.Infow("Wait for coin to appear in wallets")
	for _, n := range nodes[:len(nodes)-1] {
		if err := n.client.HasFunds(); err != nil {
			return err
		}
	}

	log.Infow("Connecting peers")
	for i, n := range nodes[:len(nodes)-1] {
		err = n.client.Connect(nodes[i+1].client.Key(), nodes[i+1].host)
		if err != nil {
			return err
		}

	}

	// Open channels. Because the sender will always choose the channel with
	// the highest balance, the channels will be utilized roughly equally.
	log.Infow("Open channels", "channel_count", cfg.Channels, "capacity_sat", cfg.ChannelCapacitySat)
	for i := 0; i < cfg.Channels; i++ {
		for i, n := range nodes[:len(nodes)-1] {
			err := n.client.OpenChannel(nodes[i+1].client.Key(), cfg.ChannelCapacitySat)
			if err != nil {
				return err
			}
		}
	}

	log.Infow("Confirm channels")
	_, err = bitcoindConn.GenerateToAddress(6, addr, nil)
	if err != nil {
		return err
	}

	log.Infow("Waiting for channels to become active")
	for i, n := range nodes {
		expectedChannels := cfg.Channels
		if i > 0 && i < len(nodes)-1 {
			expectedChannels *= 2
		}

		for {
			activeChannels, err := n.client.ActiveChannels()
			if err != nil {
				return err
			}
			if activeChannels == expectedChannels {
				break
			}
			time.Sleep(time.Second)
		}
	}

	log.Infow("Channels active")
	return nil
}
