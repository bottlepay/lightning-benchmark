package main

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/niftynei/glightning/glightning"
)

type clightningConnection struct {
	client *glightning.Lightning

	key string
}

func getClightningConnection(cfg *clightningConfig) (*clightningConnection, error) {
	logger := log.With("host", cfg.RpcHost)

	client := glightning.NewLightning()
	client.StartUp(cfg.RpcHost)

	logger.Infow("Attempting to connect to c-lightning (please be patient)")
	var key string
	for {
		info, err := client.GetInfo()
		if err == nil {
			if !info.IsBitcoindSync() || !info.IsLightningdSync() {
				time.Sleep(time.Second)

				continue
			}

			key = info.Id

			logger.Infow("Connected to c-lightning", "key", info.Id)
			break
		}

		time.Sleep(time.Second)
	}

	return &clightningConnection{
		client: client,
		key:    key,
	}, nil
}

func (l *clightningConnection) Close() {
}

func (l *clightningConnection) Key() string {
	return l.key
}

func (l *clightningConnection) Connect(key, address string) error {
	parts := strings.Split(address, ":")
	host := parts[0]
	port, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return err
	}

	_, err = l.client.ConnectPeer(key, host, uint(port))
	return err
}

func (l *clightningConnection) NewAddress() (string, error) {
	return l.client.NewAddr()
}

func (l *clightningConnection) OpenChannel(peerKey string, amtSat int64) error {
	sat := glightning.NewSat64(uint64(amtSat))
	_, err := l.client.FundChannel(peerKey, sat)
	return err
}

func (l *clightningConnection) ActiveChannels() (int, error) {
	channels, err := l.client.ListChannelsBySource("")
	if err != nil {
		return 0, err
	}

	var activeCount int
	for _, ch := range channels {
		if ch.IsActive {
			activeCount++
		}
	}

	// Both channel ends are represented in the list.
	return activeCount / 2, nil
}

func (l *clightningConnection) AddInvoice(amtMsat int64) (string, error) {
	label := randomString(20)
	invoice, err := l.client.Invoice(uint64(amtMsat), label, "test")
	if err != nil {
		return "", err
	}
	return invoice.Bolt11, nil
}

func (l *clightningConnection) SendPayment(invoice string) error {
	status, err := l.client.PayBolt(invoice)
	if err != nil {
		return err
	}

	if status.Status != "complete" {
		return errors.New("payment failed")
	}

	return nil
}

func (l *clightningConnection) SendKeysend(destination string, amtMsat int64) error {
	return errors.New("not implemented")
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func (l *clightningConnection) HasFunds() error {
	for {
		resp, err := l.client.ListFunds()
		if err != nil {
			return err
		}
		if len(resp.Outputs) > 0 {
			return nil
		}

		time.Sleep(time.Second)
	}
}
