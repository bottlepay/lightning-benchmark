# Node benchmark

This repository contains a load test for lightning node software.

The test creates a setup with a bitcoind regtest node and two connected
lightning nodes instances.

A test application spins up 100 workers that continously request an invoice
from one instance and pay it from the other one.

Output is the number of transaction that are settled per second (TPS) and the
average payment latency.

## How to run

`./run.sh <configuration>`

The following configurations are available:

Configuration | Implementation | Backend | Channels | Workers | Options
---|---|---|---|---|--
`lnd-bbolt` | lnd 0.12.1 | bbolt | 10 | 100  |
`lnd-bbolt-keysend` | lnd 0.12.1  | bbolt | 10 | 100 | keysend
`lnd-etcd` | lnd 0.12.1  | single etcd instance | 10 | 100 |
`lnd-etcd-cluster` | lnd 0.12.1  | three-instance etcd cluster | 10 | 100  |
`clightning` | c-lightning 0.9.3 | sqlite | 1 <sup>[1]</sup> | 100 |
`eclair` | eclair 0.6.1 | sqlite | 10 | 100 |
`eclair-postgres` | eclair 0.6.1 | postgres | 10 | 100 |

<sup>1</sup> Multiple channels are not supported in c-lightning  

## Results

Below are the test results after 10,000 payments on the following machine:

* Google Cloud `n2d-standard-8` instance (8 vCPUs, 32 GB memory)
* 100 GB zonal pd-ssd with ext4 filesystem
* Ubuntu 20.04 LTS

| Configuration | Transactions / sec | Avg latency (sec) |
|--|--|--|
|`eclair`| 89 | 1.1 |
|`eclair-postgres`| 46 | 2.1 |
|`clightning`| 61 | 1.6  |
|`lnd-bbolt-keysend`| 35 | 2.8 |
|`lnd-bbolt`| 33 | 3.0 |
|`lnd-etcd`| 4 | 29.2 |
|`lnd-etcd-cluster`| 4 | 31.8 |

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

## Use vault

### Vault setup

* Start docker-compose in the `vault` dir: `docker-compose up -d`
* Copy vault plugin out of container: `docker-compose cp lndsigner:/bin/vault-plugin-lndsigner vault_plugins/`
* Download minio client: wget https://dl.min.io/client/mc/release/linux-amd64/mc
* Define alias for this minio instance: `mc alias set local http://127.0.0.1:19000 ROOTUSER CHANGEME123`
* Create bucket `vault-storage`: `mc mb local/vault-storage`
* Restart `vault-server` container. Now that the bucket exists, it can start up properly.
* Log into the vault server: `docker-compose exec vault-server sh`
* Set the vault address: `export VAULT_ADDR=http://127.0.0.1:8200`
* Init vault: `vault operator init -key-shares=1 -key-threshold=1`
* Save token and unseal key
* Unseal the vault with the unseal key: `vault operator unseal`
* Set the token variable: `export VAULT_TOKEN=<token here>`
* Get plugin sha256 hash for use in the next step: `sha256sum /vault/plugins/vault-plugin-lndsigner`
* Register plugin: `vault plugin register --sha256 <sha256> secret vault-plugin-lndsigner`
* Enable plugin: `vault secrets enable --path=lndsigner vault-plugin-lndsigner`
* Create node: `vault write lndsigner/lnd-nodes network=regtest`
* Update `signer.conf` with node key
* Update `VAULT_TOKEN` in `vault/docker-compose.yml` with vault token
* Restart lndsigner `docker-compose up -d`
* `docker-compose logs -f lndsigner` should show that it is running properly

### Run test

* Go back to the root directory
* Make sure nothing is running: `docker-compose -f docker-compose-bbolt.yml down -v --remove-orphans`
* Start just Alice: `docker-compose -f docker-compose-bbolt.yml up -d lnd-alice`
* Shell into Alice: `docker-compose -f docker-compose-bbolt.yml exec lnd-alice bash`
* Create wallet: `lncli --network=regtest --tlscertpath=/cfg/tls.cert --macaroonpath=/cfg/admin.macaroon createwatchonly /lndsigner/accounts.json`
* Now run the benchmark: `./run.sh lnd-bbolt`
* Look at the `lndsigner` logs and see that it is working.