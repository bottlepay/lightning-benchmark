# Node benchmark

This repository contains a load test for lightning node software.

The test creates a setup with a bitcoind regtest node and two connected
lightning nodes instances.

A test application spins up 100 goroutines that continously request an invoice
from one instance and pay it from the other one.

Output is the number of transaction that are settled per second (TPS).

## How to run

### LND

With bbolt database backend: `./run.sh docker-compose-bbolt.yml`

With etcd database backend: `./run.sh docker-compose-etcd.yml`

The lnd test runs with 10 channels between both nodes.

### c-lightning

`./run.sh docker-compose-clightning.yml`

C-lightning does not support multiple channels between the nodes. The test runs
with a single channel.

## Configuration

In the file `loadtest.yml` the following test parameters can be set:
* `paymentAmountMsat`: the test amount that is paid
* `processes`: the number of parallel processes