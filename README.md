# Node benchmark

This repository contains a load test for `lnd`.

The test creates a setup with a bitcoind regtest node and two lnd instances that
share a single channel between them. 

A test application spins up 100 goroutines that continously request an invoice
from one instance and pay it from the other one.

Output is the number of transaction that are settled per second (TPS).

## How to run

`docker-compose down -v && docker-compose up --build | grep loadtest`

The `down` command makes sure that all leftovers from a previous run are
removed. `up` will then spin up the stack and output logs as a foreground
process. Grep filters the output to only show the test results.