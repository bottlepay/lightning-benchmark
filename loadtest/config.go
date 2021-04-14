package main

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type lndConfig struct {
	TlsCertPath  string `yaml:"tlsCertPath"`
	RpcHost      string `yaml:"rpcHost"`
	MacaroonPath string `yaml:"macaroonPath"`
}

type clightningConfig struct {
	RpcHost string `yaml:"rpcHost"`
}

type clientConfig struct {
	Lnd        *lndConfig        `yaml:"lnd"`
	Clightning *clightningConfig `yaml:"clightning"`
	Eclair     *eclairConfig     `yaml:"eclair"`

	Host string `yaml:"host"`
}

type bitcoindConfig struct {
	Host string
	User string
	Pass string
}

type eclairConfig struct {
	RpcHost  string `yaml:"rpcHost"`
	Password string `yaml:"password"`
}

type config struct {
	Sender                 clientConfig   `yaml:"sender"`
	Receiver               clientConfig   `yaml:"receiver"`
	Bitcoind               bitcoindConfig `yaml:"bitcoind"`
	PaymentAmountMsat      int64          `yaml:"paymentAmountMsat"`
	Connections            int
	ProcessesPerConnection int `yaml:"processesPerConnection"`
	Channels               int
	ChannelCapacitySat     int64 `yaml:"channelCapacitySat"`
	Keysend                bool
}

func loadConfig() (*config, error) {
	yamlFile, err := ioutil.ReadFile("loadtest.yml")
	if err != nil {
		return nil, err
	}

	var cfg config
	err = yaml.UnmarshalStrict(yamlFile, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Connections == 0 {
		return nil, errors.New("connections must be set")
	}

	if cfg.ProcessesPerConnection == 0 {
		return nil, errors.New("processes per connection must be set")
	}

	if cfg.PaymentAmountMsat == 0 {
		return nil, errors.New("payment amount must be set")
	}

	if cfg.Channels == 0 {
		return nil, errors.New("channels must be set")
	}

	if cfg.ChannelCapacitySat == 0 {
		return nil, errors.New("channel capacity must be set")
	}

	return &cfg, nil
}
