#!/bin/sh

echo "Running integration tests ..."
pwd

docker-compose -f docker-compose.yaml -f docker-compose.test.yaml build
docker-compose -f docker-compose.yaml -f docker-compose.test.yaml  up --exit-code-from test