#!/usr/bin/env bash

set -euo pipefail

make docker-build

docker run --publish 127.0.0.1:8080:8080 \
           --name gatus \
           --volume `pwd`/gatus_configuration:/config:ro \
           --detach \
           twinproduction/gatus:latest
