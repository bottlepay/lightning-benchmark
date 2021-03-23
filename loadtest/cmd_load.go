package main

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

const (
	// The number of concurrent payment processes.
	concurrency = 100
)

var settledChan = make(chan struct{})

var loadCommand = cli.Command{
	Name:   "load",
	Action: load,
}

func load(_ *cli.Context) error {
	yamlFile, err := ioutil.ReadFile("loadtest.yml")
	if err != nil {
		return err
	}

	var cfg config
	err = yaml.UnmarshalStrict(yamlFile, &cfg)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for t := 0; t < concurrency; t++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := loadThread(&cfg.Sender, &cfg.Receiver)
			if err != nil {
				log.Errorw("Send error", "err", err)
				os.Exit(1)
			}
		}()
	}

	const statBlockSize = 100
	go func() {
		last := time.Now()
		settledCount := 0
		for {
			<-settledChan
			settledCount++

			if settledCount%statBlockSize == 0 {
				now := time.Now()
				tps := float64(statBlockSize) / now.Sub(last).Seconds()
				log.Infow("Speed", "tps", tps, "count", settledCount)

				last = now
			}
		}
	}()

	wg.Wait()

	return nil
}

func loadThread(senderCfg *clientConfig, receiverCfg *clientConfig) error {
	senderConn, err := getClientConn(&senderCfg.Lnd)
	if err != nil {
		return err
	}
	defer senderConn.Close()
	senderClient := routerrpc.NewRouterClient(senderConn)

	receiverConn, err := getClientConn(&receiverCfg.Lnd)
	if err != nil {
		return err
	}
	defer receiverConn.Close()
	receiverClient := lnrpc.NewLightningClient(receiverConn)

	send := func() error {
		addResp, err := receiverClient.AddInvoice(context.Background(), &lnrpc.Invoice{
			Value: 1,
		})
		if err != nil {
			return err
		}
		invoice := addResp.PaymentRequest

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stream, err := senderClient.SendPayment(ctx, &routerrpc.SendPaymentRequest{
			PaymentRequest:    invoice,
			TimeoutSeconds:    60,
			NoInflightUpdates: true,
		})
		if err != nil {
			log.Errorw("Error sending payment", "err", err)
			return err
		}

		update, err := stream.Recv()
		if err != nil {
			return err
		}

		if update.State != routerrpc.PaymentState_SUCCEEDED {
			return errors.New("payment failed")
		}

		settledChan <- struct{}{}

		return nil
	}

	for {
		err := send()
		if err != nil {
			return err
		}
	}
}
