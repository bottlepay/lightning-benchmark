# Node benchmark

This repository contains a load test for `lnd`.

The test creates a setup with a bitcoind regtest node and two lnd instances that
share a 10 channels between them. 

A test application spins up 100 goroutines that continously request an invoice
from one instance and pay it from the other one.

Output is the number of transaction that are settled per second (TPS).

## How to run

With bbolt database backend: `./run.sh docker-compose-bbolt.yml`

With etcd database backend: `./run.sh docker-compose-etcd.yml`

## Configuration

In the file `loadtest.yml` the following test parameters can be set:
* `paymentAmountMsat`: the test amount that is paid
* `processes`: the number of parallel processes