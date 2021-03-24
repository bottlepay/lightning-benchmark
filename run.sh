#!/bin/bash

if [[ $1 == "" ]]
then
  echo "usage: run.sh <docker compose file>"
  exit 0
fi

# Make sure that all leftovers from a previous run are removed. 
docker-compose -f $1 down -v --remove-orphans

# Spin up the stack and output logs as a foreground process. Grep filters the
# output to only show the test results.
docker-compose -f $1 up --build | grep loadtest
