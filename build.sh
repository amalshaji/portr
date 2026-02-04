#!/bin/bash

cd /Users/reggie.pierce/Projects/github-regbo/portr/tunnel
go build -o ./bin/portr ./cmd/portr
go build -o ./bin/portrd ./cmd/portrd