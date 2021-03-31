module github.com/bottlepay/loadtest

go 1.13

require (
	github.com/btcsuite/btcd v0.21.0-beta.0.20210316172410-f86ae60936d7
	github.com/btcsuite/btcutil v1.0.2
	github.com/lightningnetwork/lnd v0.11.1-beta.rc4
	github.com/niftynei/glightning v0.8.2
	github.com/urfave/cli v1.22.4
	go.uber.org/zap v1.14.1
	google.golang.org/grpc v1.27.0
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/yaml.v2 v2.2.3
)

replace github.com/niftynei/glightning => github.com/joostjager/glightning v0.8.3-0.20210325135629-f1548ac8aeb8
