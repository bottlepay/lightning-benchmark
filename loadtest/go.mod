module github.com/bottlepay/loadtest

go 1.13

require (
	github.com/btcsuite/btcd v0.21.0-beta.0.20210513141527-ee5896bad5be
	github.com/btcsuite/btcutil v1.0.3-0.20210527170813-e2ba6805a890
	github.com/gorilla/websocket v1.4.2
	github.com/lightningnetwork/lnd v0.13.0-beta
	github.com/niftynei/glightning v0.8.2
	github.com/urfave/cli v1.22.4
	go.uber.org/zap v1.14.1
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/yaml.v2 v2.2.3
)

replace github.com/niftynei/glightning => github.com/joostjager/glightning v0.8.3-0.20210325135629-f1548ac8aeb8

replace go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
