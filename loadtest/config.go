package main

type lndConfig struct {
	TlsCertPath  string `yaml:"tlsCertPath"`
	Host         string `yaml:"host"`
	RpcHost      string `yaml:"rpcHost"`
	MacaroonPath string `yaml:"macaroonPath"`
}

type clientConfig struct {
	Lnd lndConfig `yaml:"lnd"`
}

type bitcoindConfig struct {
	Host string
	User string
	Pass string
}

type config struct {
	Sender   clientConfig   `yaml:"sender"`
	Receiver clientConfig   `yaml:"receiver"`
	Bitcoind bitcoindConfig `yaml:"bitcoind"`
}
