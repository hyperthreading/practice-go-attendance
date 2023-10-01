#!/bin/bash

set -x

while true; do
    time go mod download
    time go build -o ./bin/_test_api ./cmd/api
    
    inotifywait -e modify,create,delete,move -r ./cmd ./internal ./go.mod ./go.sum
done