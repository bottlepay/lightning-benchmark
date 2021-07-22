package main

import "errors"

type nodeInterface interface {
	Key() string
	Connect(key, host string) error
	NewAddress() (string, error)
	OpenChannel(peerKey string, amtSat int64) error
	ActiveChannels() (int, error)
	AddInvoice(amtMsat int64) (string, error)
	SendPayment(invoice string) error
	SendKeysend(destination string, amtMsat int64) error
	Close()
	HasFunds() error
}

type info struct {
	key    string
	synced bool
}

func getNodeConnection(cfg *clientConfig) (nodeInterface, error) {
	switch {
	case cfg.Lnd != nil:
		return getLndConnection(cfg.Lnd)

	case cfg.Clightning != nil:
		return getClightningConnection(cfg.Clightning)

	case cfg.Eclair != nil:
		return getEclairConnection(cfg.Eclair)
	}

	return nil, errors.New("unrecognized config")
}
