# Node benchmark

This repository contains a load test for lightning node software.

The test creates a setup with a bitcoind regtest node and two connected
lightning nodes instances.

A test application spins up 100 workers that continously request an invoice
from one instance and pay it from the other one.

Output is the number of transaction that are settled per second (TPS).

## How to run

`./run.sh <configuration>`

The following configurations are available:

Configuration | Implementation | Backend | Channels | Workers | Options
---|---|---|---|---|--
`lnd-bbolt` | lnd | bbolt | 10 | 100  |
`lnd-bbolt-keysend` | lnd | bbolt | 10 | 100 | keysend
`lnd-etcd` | lnd | single etcd instance | 10 | 100 |
`lnd-etcd-cluster` | lnd | three-instance etcd cluster | 10 | 100  |
`clightning` | c-lightning | sqlite | 1 <sup>[1]</sup> | 100 |
`eclair` | eclair | sqlite | 10 | 10  <sup>[2]</sup>|

<sup>1</sup> Multiple channels are not supported in c-lightning  
<sup>2</sup> Reduced number of workers to prevent timeouts

## Results

Below are the test results after 10,000 payments on the following machine:

* Google Cloud `n2d-standard-8` instance (8 vCPUs, 32 GB memory)
* 100 GB zonal pd-ssd with ext4 filesystem
* Ubuntu 20.04 LTS

| Configuration | Transactions per second |
|--|--|
|`clightning`| 61  |
|`lnd-bbolt-keysend`| 35  |
|`lnd-bbolt`| 33  |
|`eclair`| 12 |
|`lnd-etcd`| 4 |
|`lnd-etcd-cluster`| 4 |

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