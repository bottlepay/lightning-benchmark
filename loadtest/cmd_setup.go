package main

import (
	"io/ioutil"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/urfave/cli"
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
	senderClient, err := getLndConnection(&cfg.Sender.Lnd)
	if err != nil {
		return err
	}
	defer senderClient.Close()
	addrResp, err := senderClient.NewAddress()
	if err != nil {
		return err
	}
	senderAddr, err := btcutil.DecodeAddress(addrResp, &chaincfg.RegressionNetParams)
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

	receiverClient, err := getLndConnection(&cfg.Receiver.Lnd)
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
	err = senderClient.Connect(receiverKey, cfg.Receiver.Lnd.Host)
	if err != nil {
		return err
	}

	log.Infow("Open channel")
	err = senderClient.OpenChannel(receiverKey, 10000000)
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
		activeChannels, err := senderClient.HasActiveChannels()
		if err != nil {
			return err
		}
		if activeChannels {
			break
		}
		time.Sleep(time.Second)
	}

	log.Infow("Channel active")
	return nil
}
