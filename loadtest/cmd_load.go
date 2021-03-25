package main

import (
	"os"
	"sync"
	"time"

	"github.com/urfave/cli"
)

var settledChan = make(chan struct{})

var loadCommand = cli.Command{
	Name:   "load",
	Action: load,
}

func load(_ *cli.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	log.Infow("Starting payment processes",
		"process_count", cfg.Processes, "amt_msat", cfg.PaymentAmountMsat)

	var wg sync.WaitGroup
	for t := 0; t < cfg.Processes; t++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := loadThread(&cfg.Sender, &cfg.Receiver, cfg.PaymentAmountMsat)
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

func loadThread(senderCfg *clientConfig, receiverCfg *clientConfig, amtMsat int64) error {
	senderClient, err := getNodeConnection(senderCfg)
	if err != nil {
		return err
	}
	defer senderClient.Close()

	receiverClient, err := getNodeConnection(receiverCfg)
	if err != nil {
		return err
	}
	defer receiverClient.Close()

	send := func() error {
		invoice, err := receiverClient.AddInvoice(amtMsat)
		if err != nil {
			return err
		}

		err = senderClient.SendPayment(invoice)
		if err != nil {
			log.Errorw("Error sending payment", "err", err)
			return err
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
