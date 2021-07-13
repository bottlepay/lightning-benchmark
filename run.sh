#!/bin/bash

if [[ $1 == "" ]]
then
  echo "usage: run.sh lnd-bbolt | lnd-bbolt-keysend | lnd-etcd | lnd-etcd-cluster | clightning | eclair"
  exit 0
fi

case $1 in
  "lnd-bbolt")
    DOCKER_COMPOSE_FILE=docker-compose-bbolt.yml
    export LOADTEST_CONFIG_FILE=loadtest-lnd.yml
    ;;

 "lnd-bbolt-keysend")
    DOCKER_COMPOSE_FILE=docker-compose-bbolt.yml
    export LOADTEST_CONFIG_FILE=loadtest-lnd-keysend.yml
    ;;

  "lnd-etcd")
    DOCKER_COMPOSE_FILE=docker-compose-etcd.yml
    export LOADTEST_CONFIG_FILE=loadtest-lnd.yml
    ;;

  "lnd-etcd-cluster")
    DOCKER_COMPOSE_FILE=docker-compose-etcd-cluster.yml
    export LOADTEST_CONFIG_FILE=loadtest-lnd.yml
    ;;

  "clightning")
    DOCKER_COMPOSE_FILE=docker-compose-clightning.yml
    export LOADTEST_CONFIG_FILE=loadtest-clightning.yml
    ;;

  "eclair")
    DOCKER_COMPOSE_FILE=docker-compose-eclair.yml
    export LOADTEST_CONFIG_FILE=loadtest-eclair.yml
    ;;

  "eclair-postgres")
    DOCKER_COMPOSE_FILE=docker-compose-eclair-postgres.yml
    export LOADTEST_CONFIG_FILE=loadtest-eclair.yml
    ;;

  *)
    echo "unknown configuration"
    exit 1
    ;;
esac

echo DOCKER_COMPOSE_FILE: $DOCKER_COMPOSE_FILE
echo LOADTEST_CONFIG_FILE: $LOADTEST_CONFIG_FILE

# Make sure that all leftovers from a previous run are removed. 
docker-compose -f $DOCKER_COMPOSE_FILE down -v --remove-orphans

# Spin up the stack and output logs as a foreground process. Grep filters the
# output to only show the test results.
docker-compose -f $DOCKER_COMPOSE_FILE up --build | grep loadtest
