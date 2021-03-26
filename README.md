# Node benchmark

This repository contains a load test for lightning node software.

The test creates a setup with a bitcoind regtest node and two connected
lightning nodes instances.

A test application spins up 100 goroutines that continously request an invoice
from one instance and pay it from the other one.

Output is the number of transaction that are settled per second (TPS).

## How to run

`./run.sh <configuration>`

The following configurations are available:

Configuration | Description
---|---
`lnd-bbolt` | `lnd` with a bbolt database backend, 10 channels between the nodes
`lnd-bbolt-keysend` | `lnd` with a bbolt database backend, 10 channels between the nodes, spontaneous keysend payments
`lnd-etcd` | `lnd` with a single etcd instance as the database backend, 10 channels between the nodes
`lnd-etcd-cluster` | `lnd` with a three-instance etcd cluster as the database backend, 10 channels between the nodes
`clightning` | `c-lightning`, single channel between the nodes (multiple channels not supported)

## Profiling

For `lnd` nodes, a cpu profile can be extracted for further analysis. The sender node profiler is reachable through port 5000 on the host. The receiver node profiler is available at port 5001.

To display the profile in a browser, run:

`go tool pprof -http 0.0.0.0:7777 http://localhost:5000/debug/pprof/profile`

## Configuration

The `loadtest` container reads test parameters from the file `loadtest.yml` the
following parameters are available:

* `paymentAmountMsat`: the test amount that is paid
* `processes`: the number of parallel processes
* `channels`: the number of channels between the two test nodes
* `channelCapacitySat`: capacity of the channel(s)