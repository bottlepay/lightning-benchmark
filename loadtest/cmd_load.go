package main

import (
	"os"
	"sync"
	"time"

	"github.com/urfave/cli"
)

var settledChan = make(chan time.Duration)

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

			err := loadThread(
				&cfg.Sender, &cfg.Receiver,
				cfg.PaymentAmountMsat, cfg.Keysend,
			)
			if err != nil {
				log.Errorw("Send error", "err", err)
				os.Exit(1)
			}
		}()
	}

	const statBlockSize = 1000
	go func() {
		settledCount := 0
		for {
			last := time.Now()
			var totalTime time.Duration
			for i := 0; i < statBlockSize; i++ {
				totalTime += <-settledChan
				settledCount++
			}

			now := time.Now()
			tps := float64(statBlockSize) / now.Sub(last).Seconds()
			latency := totalTime.Seconds() / statBlockSize
			log.Infow("Speed",
				"tps", tps,
				"count", settledCount,
				"avg_latency_sec", latency)

			last = now
		}
	}()

	wg.Wait()

	return nil
}

func loadThread(senderCfg *clientConfig, receiverCfg *clientConfig,
	amtMsat int64, keysend bool) error {

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

	var send func() error
	if keysend {
		receiverKey := receiverClient.Key()

		send = func() error {
			return senderClient.SendKeysend(receiverKey, amtMsat)
		}
	} else {
		send = func() error {
			invoice, err := receiverClient.AddInvoice(amtMsat)
			if err != nil {
				return err
			}

			return senderClient.SendPayment(invoice)
		}
	}
	for {
		start := time.Now()
		err := send()
		if err != nil {
			log.Errorw("Error sending payment", "err", err)
			return err
		}

		settledChan <- time.Since(start)
	}
}
